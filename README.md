# dvid-transfer
DVID-to-DVID data transfer via APIs

## Build

     % go build
     
## Run

     % dvid-transfer http://dvid1:7000/api/node/03fc/labels http://dvid2:9000/api/node/8f3d/labels

This transfers all data from labelblk instance "labels" from dvid1:7000, version 03fc to dvid2:9000, version 8f3d.
