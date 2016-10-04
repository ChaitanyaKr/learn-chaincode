#Instructions for contract usage

Note: This is WIP code and subject to change

Testing the contracts - Key function call examples

##Registering  (only on sandbox)
```
CORE_CHAINCODE_ID_NAME=comp CORE_PEER_ADDRESS=0.0.0.0:7051 ./compliance
CORE_CHAINCODE_ID_NAME=cont CORE_PEER_ADDRESS=0.0.0.0:7051 ./container
CORE_CHAINCODE_ID_NAME=lContr CORE_PEER_ADDRESS=0.0.0.0:7051 ./logisticsContract
```

Once the chaincodes are registered, execute script.sh from a separate terminal. The script will execute the operations given below (starting from Init). Ensure that the output of the commands matches as given here.


##1. Init 
Send version number, container and compliance contract ids in the deploy call for BillofLading.
Similarly, the container contract needs the compliance contract id.
```
./peer chaincode deploy -n lContr -c '{"function":"Init", "args":["{\"Version\":\"2.0.0\", \"containercc\":\"cont\", \"compliancecc\":\"comp\"}"]}'
./peer chaincode deploy -n cont -c '{"function":"Init", "args":["{\"Version\":\"2.0.0\", \"compliancecc\":\"comp\"}"]}'
./peer chaincode deploy -n comp -c '{"function":"Init", "args":["{\"Version\":\"2.0.0\"}"]}'
```
##2. Create Bill of Lading registration (note : it checks for duplicate B/L)
 
```
Example: 
./peer chaincode invoke -n lContr -c '{"function":"registerBillOfLading", "args":["{\"blno\":\"1020310\", \"containernos\":\"MSKU12344321,MAER909090\", \"hazmat\":false, \"mintemperature\":-10, \"maxtemperature\":30, \"minhumidity\":0, \"maxhumidity\":50, \"minlight\":0, \"maxlight\":30, \"minacceleration\":0.01, \"maxacceleration\":2}"]}'
```
This causes the following records to be created 
```
- container record (mapping between B/L and container) (Note: If container number already exists in stub, container contract assumes it is for a different B/L that's completed transit, so it takes that record and puts it back in stub with the BL number appended tocontainer number. Then it creates the new container record)
{"containerno":"MSKU12344321","blno":"1020310","timestamp":"2016-05-06 05:40:41.780399446 +0000 UTC"}
{"containerno":"MAER909090","blno":"1020310","timestamp":"2016-05-06 05:40:41.780399446 +0000 UTC"}

-compliance record (B/L compliance record)
{"blno":"1020310","type":"SHIPPING","compliance":true,"assetalerts":null,"active":true,"timestamp":"2016-05-06 05:40:41.780399446 +0000 UTC"}

- bill of lading registration : 
{"blno":"1020310", "containernos":"MSKU12344321,MAER909090", "hazmat":false, "mintemperature":-10, "maxtemperature":30, "minhumidity":0, "maxhumidity":50, "minlight":0, "maxlight":30, "minacceleration":0.01, "maxacceleration":2}
```
These can be queried as explained in steps 4 and 5 below

##3. Update container data
```
./peer chaincode invoke -n cont -c '{"function":"updateContainerLogistics", "args":["{\"containerno\":\"MAER909090\",\"location\":{\"latitude\":10, \"longitude\":9}, \"temperature\":41}"]}' 
```
##4. read container current status
```
./peer chaincode query -n cont -c '{"function":"readContainerCurrentStatus", "args":["{\"containerno\":\"MAER909090\"}"]}' 
```
This returns: 
```
{"containerno":"MAER909090","blno":"1020310","location":{"latitude":10,"longitude":9},"temperature":41,"timestamp":"2016-05-06 05:40:41.780399446 +0000 UTC","alerts":"{\"tempalert\":\"above\"}"}
```
##5. read compliance data
```
Example:
./peer chaincode query -n comp -c '{"function":"readCurrentComplianceState", "args":["{\"blno\":\"1020310\"}"]}'
```

This returns: 
```
{"blno":"1020310","type":"SHIPPING","compliance":false,"assetalerts":{"MAER909090":"{\"tempalert\":\"above\"}"},"active":true,"timestamp":"2016-05-06 05:40:41.780399446 +0000 UTC"}
```
##6. compliance history
```
./peer chaincode query -n comp -c '{"function":"readComplianceHistory", "args":["{\"blno\":\"1020310\"}"]}'
```
This returns: 

```
{"comphistory":["{\"blno\":\"1020310\",\"type\":\"SHIPPING\",\"compliance\":false,\"assetalerts\":{\"MAER909090\":\"{\\\"tempalert\\\":\\\"above\\\"}\"},\"active\":true,\"timestamp\":\"2016-05-06 05:40:41.780399446 +0000 UTC\"}","{\"blno\":\"1020310\",\"type\":\"SHIPPING\",\"compliance\":true,\"assetalerts\":null,\"active\":true,\"timestamp\":\"2016-05-06 05:40:41.780399446 +0000 UTC\"}"]}
```
Note: Most recent record is on top
