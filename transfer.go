package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type BaseMetadata struct {
	TypeName    string
	TypeURL     string
	TypeVersion string
	Name        string
	RepoUUID    string
	Compression string
	Checksum    string
	Persistence string
	Versioned   bool
}

type Metadata struct {
	Base     BaseMetadata
	Extended interface{}
}

func getMetadata(baseUrl string) *Metadata {
	infoUrl := baseUrl + "/info"
	resp, err := http.Get(infoUrl)
	if err != nil {
		fmt.Printf("Error on getting metadata: %s\n", err.Error())
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Bad status on getting metadata: %d\n", resp.StatusCode)
		os.Exit(1)
	}
	metadata, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Could not read metadata: %s\n", err.Error())
		os.Exit(1)
	}
	m := new(Metadata)
	if err := json.Unmarshal(metadata, m); err != nil {
		fmt.Printf("Error parsing metadata: %s\n", err.Error())
		os.Exit(1)
	}
	return m
}

type LabelMetadata struct {
	Base     BaseMetadata
	Extended struct {
		BlockSize [3]int
		MinIndex  [3]int
		MaxIndex  [3]int
	}
}

func sendLabels(src, dst *Metadata, srcUrl, dstUrl string) {
	// Get the index extents for src.
	infoUrl := srcUrl + "/info"
	resp, err := http.Get(infoUrl)
	if err != nil {
		fmt.Printf("Error on getting metadata: %s\n", err.Error())
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Bad status on getting metadata: %d\n", resp.StatusCode)
		os.Exit(1)
	}
	metadata, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Could not read metadata: %s\n", err.Error())
		os.Exit(1)
	}
	m := new(LabelMetadata)
	if err := json.Unmarshal(metadata, m); err != nil {
		fmt.Printf("Error parsing label metadata: %s\n", err.Error())
		os.Exit(1)
	}
	minIndex := m.Extended.MinIndex
	maxIndex := m.Extended.MaxIndex
	blockSize := m.Extended.BlockSize
	if blockSize[0] != blockSize[1] {
		fmt.Printf("Can't handle non-cubic block sizes: %v\n", blockSize)
	}
	fmt.Printf("MinIndex: %v\n", minIndex)
	fmt.Printf("MaxIndex: %v\n", maxIndex)

	// Iterate through each XY plane of one block deep data.
	// Send it to destination, synchronously.
	nx := maxIndex[0] - minIndex[0] + 1
	ny := maxIndex[1] - minIndex[1] + 1

	// # voxels in x, and xy
	vx := nx * blockSize[0]
	vy := ny * blockSize[1]
	vz := blockSize[2]

	strips := 1
	if vx*vy*vz*8 > 2000000000 {
		strips = (vx*vy*vz*8)/2000000000 + 1
	}
	byPerStrip := ny / strips
	fmt.Printf("Strips per layer: %d\n", strips)

	ox := minIndex[0] * blockSize[0]
	for z := minIndex[2]; z <= maxIndex[2]; z++ {
		oz := z * blockSize[2]
		by0 := minIndex[1]
		for n := 0; n < strips; n++ {
			if by0 > maxIndex[1] {
				break
			}
			by1 := by0 + byPerStrip - 1
			if by1 > maxIndex[1] {
				by1 = maxIndex[1]
			}

			oy := by0 * blockSize[1]
			vy := (by1 - by0 + 1) * blockSize[1]
			url := fmt.Sprintf("%s/raw/0_1_2/%d_%d_%d/%d_%d_%d", srcUrl,
				vx, vy, vz, ox, oy, oz)
			url2 := fmt.Sprintf("%s/raw/0_1_2/%d_%d_%d/%d_%d_%d", dstUrl,
				vx, vy, vz, ox, oy, oz)
			fmt.Printf("Transfering: %s -> %s\n", url, url2)
			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("Receive error: %s\n", err.Error())
				os.Exit(1)
			}
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Bad status on receiving data: %d\n", resp.StatusCode)
				os.Exit(1)
			}
			resp2, err := http.Post(url2, "application/octet-stream", resp.Body)
			if err != nil {
				fmt.Printf("Transmit error: %s\n", err.Error())
				os.Exit(1)
			}
			if resp2.StatusCode != http.StatusOK {
				fmt.Printf("Bad status on sending data: %d\n", resp2.StatusCode)
				os.Exit(1)
			}

			by0 += byPerStrip
		}
	}
}

func sendROI(src, dst *Metadata, srcUrl, dstUrl string) {
	url := srcUrl + "/roi"
	url2 := dstUrl + "/roi"
	fmt.Printf("Transfering: %s -> %s\n", url, url2)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Receive error: %s\n", err.Error())
		os.Exit(1)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		fmt.Printf("Bad status on receiving data: %d\n", resp.StatusCode)
		os.Exit(1)
	}
	resp2, err := http.Post(url2, "application/json", resp.Body)
	if err != nil {
		fmt.Printf("Transmit error: %s\n", err.Error())
		os.Exit(1)
	}
	if resp2.StatusCode != http.StatusOK {
		fmt.Printf("Bad status on sending data: %d\n", resp2.StatusCode)
		os.Exit(1)
	}
}

func transferData(src, dst string) {
	srcMetadata := getMetadata(src)
	dstMetadata := getMetadata(dst)
	srctype := srcMetadata.Base.TypeName
	dsttype := dstMetadata.Base.TypeName

	switch srctype {
	case "grayscale8", "uint8blk":
		if dsttype != "uint8blk" {
			fmt.Printf("Can't transfer %s to %s, need uint8blk destination\n", srctype, dsttype)
			os.Exit(1)
		}

	case "labels64":
		if dsttype != "labelblk" {
			fmt.Printf("Can't transfer %s to %s, need labelblk destination\n", srctype, dsttype)
			os.Exit(1)
		}
		sendLabels(srcMetadata, dstMetadata, src, dst)

	case "roi":
		if dsttype != "roi" {
			fmt.Printf("Can't transfer %s to %s, need roi destination\n", srctype, dsttype)
			os.Exit(1)
		}
		sendROI(srcMetadata, dstMetadata, src, dst)

	default:
		fmt.Printf("Cannot handle source data type %s\n", srcMetadata.Base.TypeName)
		os.Exit(1)
	}
}
