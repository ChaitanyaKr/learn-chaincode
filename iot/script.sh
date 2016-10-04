#!/bin/bash
#script to automate tasks given in readme.md
#Run this after registering of chaincodes

echo "Deploying Bill of Lading chaincode"
peer chaincode deploy -n blReg -c '{"function":"Init", "args":["{\"Version\":\"1.0.0\", \"containercc\":\"cont\", \"compliancecc\":\"comp\"}"]}'

echo "Deploying Container chaincode"
peer chaincode deploy -n cont -c '{"function":"Init", "args":["{\"Version\":\"1.0.0\", \"compliancecc\":\"comp\"}"]}'

echo "Deploying Compliance chaincode"
peer chaincode deploy -n comp -c '{"function":"Init", "args":["{\"Version\":\"1.0.0\"}"]}'


echo "Creating Bill of Lading registration"
peer chaincode invoke -n blReg -c '{"function":"registerBillOfLading", "args":["{\"blno\":\"1020310\", \"containernos\":\"MSKU12344321,MAER909090\", \"hazmat\":false, \"mintemperature\":-10, \"maxtemperature\":30, \"minhumidity\":0, \"maxhumidity\":50, \"minlight\":0, \"maxlight\":30, \"minacceleration\":0.01, \"maxacceleration\":2}"]}'

echo -n "Confirm output against that given in readme and press Enter"
read userInput

echo "Updating container data:"
peer chaincode invoke -n cont -c '{"function":"updateContainerLogistics", "args":["{\"containerno\":\"MAER909090\",\"location\":{\"latitude\":10, \"longitude\":9}, \"temperature\":41}"]}' 

echo "Reading container current status:"
peer chaincode query -n cont -c '{"function":"readContainerCurrentStatus", "args":["{\"containerno\":\"MAER909090\"}"]}' 

echo -n "Confirm output against that given in readme and press Enter"
read userInput

echo "Reading compliance status:"
peer chaincode query -n comp -c '{"function":"readCurrentComplianceState", "args":["{\"blno\":\"1020310\"}"]}'

echo -n "Confirm output against that given in readme and press Enter"
read userInput

echo "Reading compliance history:"
peer chaincode query -n comp -c '{"function":"readComplianceHistory", "args":["{\"blno\":\"1020310\"}"]}'

echo -n "Confirm output against that given in readme and press Enter"
read userInput
