package main
import (
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "time"
 //  "github.com/mcuadros/go-jsonschema-generator"
"github.com/hyperledger/fabric/core/chaincode/shim"
)
const CONTRACTSTATEKEY string = "BLSTATEKEY" 


type ContractState struct {
    Version      string                        `json:"version"`
    ContainerCC  string                        `json:"containercc"`
    ComplianceCC string                        `json:"compliancecc"`
}   
//const DEFAULT_SCHEMA = "http://json-schema.org/schema#"

const MYVERSION string = "1.0.0"

 
 
type Geolocation struct {
    Latitude    float64 `json:"latitude,omitempty"`
    Longitude   float64 `json:"longitude,omitempty"`
}

// This is  optional. It stands for the 'acceptable range', say 1 degree of lat and long
// at which the container should be, before it is considered 'arrived' at 'Notified Party' location'
// If not sent in, some default could be assumed (say 1 degree or )
type NotifyRange struct {
    LatRange        float64 `json:"latrange,omitempty"`
    LongRange       float64 `json:"longrange,omitempty"`
}

// This is a logistics contract, written in the context of shipping. It tracks the progress of a Bill of Lading 
// and associated containers, and raises alerts in case of violations in expected conditions

// Assumption 1. Bill of Lading is sacrosanct - Freight forwarders may issue intermediary freight bills, but
// the original B/L is the document we trackend to end. Similarly a 'Corrected B/L' scenario is not considered

// Assumption 2. A Bill of Lading can have multiple containers attached to it. We are, for simplicity, assuming that
// the same transit rules in terms of allowed ranges in temperature, humidity etc. apply across the B/L - i.e. 
// applies to all containers attached to a Bill of Lading

// Assumption 3. A shipment may switch from one container to another in transit, for various reasons. We are assuming,
// for simplicity, that the same containers are used for end to end transit.

// Initial registration of the Bill of Lading. Sets out the constrains for B/L data and the Notification details
type BillOfLadingRegistration struct {
    BLNo                 *string                  `json:"blno"` 
    ContainerNos         *string                  `json:"containernos"`    // Comma separated container numbers - keep json simple    
    Hazmat               *bool                    `json:"hazmat,omitempty"`     // shipment hazardous ?
    MinTemperature       *float64                 `json:"mintemperature,omitempty"` //split range to min and max: Jeff's input
    MaxTemperature       *float64                 `json:"maxtemperature,omitempty"` 
    MinHumidity          *float64                 `json:"minhumidity,omitempty"` //split range to min and max: Jeff's input
    MaxHumidity          *float64                 `json:"maxhumidity,omitempty"`
    MinLight             *float64                 `json:"minlight,omitempty"` //split range to min and max: Jeff's input
    MaxLight             *float64                 `json:"maxlight,omitempty"` 
    MinAcceleration      *float64                 `json:"minacceleration,omitempty"` //split range to min and max: Jeff's input
    MaxAcceleration      *float64                 `json:"maxacceleration,omitempty"`
 //NotifyLocations      *[]Geolocation            `json:"notifylocations,omitempty"` // No implementation right now
 //NotifyRange          *NotifyRange              `json:"notifyrange,omitempty"`     // To be integrated when shipping part gets sorted out
    TransitComplete       *bool                    `json:"transitcomplete,omitempty"`
    Timestamp            *string                      `json:"timestamp"`
}

// Compliance record structure
type ComplianceState struct {
    BLNo                 *string                     `json:"blno"` 
    Type                 *string                     `json:"type"` // Default: DEFTYPE
    Compliance           *bool                       `json:"compliance"`
    AssetAlerts          *map[string]string          `json:"assetalerts"`
    Active               *bool                       `json:"active,omitempty"`
    Timestamp            *string                      `json:"timestamp"`
}

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}
var contractState = ContractState{MYVERSION, "",""}

//************* main *******************
//Create SimpleChaincode instance
//************* main *******************

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple Chaincode: %s", err)
	}
}

//************* init *******************
//Chaincode initialization
//************* init *******************

func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	var stateArg ContractState
	var err error

    if len(args) != 1 {
        return nil, errors.New("init expects one argument, a JSON string with tagged version string, container and compliance chaincode uuids")
    }
    err = json.Unmarshal([]byte(args[0]), &stateArg)
    if err != nil {
        return nil, errors.New("Version argument unmarshal failed: " + fmt.Sprint(err))
    }
    if stateArg.Version != MYVERSION {
        return nil, errors.New("Contract version " + MYVERSION + " must match version argument: " + stateArg.Version)
    }
    
    //fmt.Println("chaincodes assigned")
    if stateArg.ContainerCC=="" || stateArg.ComplianceCC =="" {
        return nil, errors.New("Container and compliance chaincode ids are mandatory")
    }
    //fmt.Println("complianceChainCode ", complianceChainCode)
    //fmt.Println("containerChainCode ", containerChainCode)
    contractStateJSON, err := json.Marshal(stateArg)
    if err != nil {
        return nil, errors.New("Marshal failed for contract state" + fmt.Sprint(err))
    }
    
    err = stub.PutState(CONTRACTSTATEKEY, contractStateJSON)
    if err != nil {
        return nil, errors.New("Contract state failed PUT to ledger: " + fmt.Sprint(err))
    }
    return nil, nil
}

// ************************************
// invoke callback mode 
// ************************************
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	// Handle different functions
    switch function {
        case "registerBillOfLading" :
            return t.registerBillOfLading(stub, args)
        case "deregisterBillOfLading" :
            return t.deregisterBillOfLading(stub, args)
        default:
            return nil, errors.New("Unknown function call to compliance : invoke")
    }
}

// ************************************
// query callback mode 
// ************************************
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	// Handle different Query functions 
    switch function {
        case "getBillOfLadingRegistration" :
            return t.getBillOfLadingRegistration(stub, args)
        case "getBillOfLadingRegistrationSchema" :
            return t.getBillOfLadingRegistrationSchema(stub, args)
        default:
            return nil, errors.New("Unknown function call to compliance : invoke")
    }
}                   

// ***********************registerBillOfLading************************

func (t *SimpleChaincode) registerBillOfLading(stub *shim.ChaincodeStub, args []string) ([]byte, error) { 
	var err error
    var blReg BillOfLadingRegistration
    var compState ComplianceState
    var contractState ContractState

    // This is where the initial regestration of the Bill of Lading takes place. It would correspond to a 
    // Bill of lading being generated in shipping and contains list of associated containers
     
    if len(args) !=1 {
        err = errors.New("Incorrect number of arguments. Expecting a single JSON string with mandatory BillofLading, Container Numbers and Hazmat")
		fmt.Println(err)
		return nil, err
	}
    jsonData:=args[0]
    
    // Marshaling the input to the BillOfLadingRegistration structure. If a value is not
    // sent in, it is defaulted as null
    initJSON := []byte(jsonData)
    fmt.Println("Input asset arg: ", jsonData)
    
    err = json.Unmarshal(initJSON, &blReg)
    if err != nil {
        //err = errors.New("Unable to unmarshal input JSON data")
		fmt.Println(err)
		return nil, err
    }
   // fmt.Println(" blDef after unmarshaling [", blDef, "]") 
   
   //  Nil check for mandatory fields. Since these are defined as pointers in the
   // struct definition, they will be unmarshalled as null (json) / nil (golang)      
     if blReg.BLNo==nil||blReg.ContainerNos==nil || blReg.Hazmat ==nil {
        err = errors.New("Bill of Lading, Container Numbers,  Hazmat flag and Timestamp are mandatory")
        fmt.Println(err)
		return nil, err
     }
   //  fmt.Println(" Trimming blanks out")
   
     fmt.Println(" Check if Bill of Lading and container numbers have been sent in correctly")
     *blReg.BLNo = strings.TrimSpace(*blReg.BLNo)
     *blReg.ContainerNos = strings.TrimSpace(*blReg.ContainerNos)
     if *blReg.BLNo=="" || *blReg.ContainerNos =="" {
        err = errors.New("Bill of Lading  / Container numbers cannot be blank")
        fmt.Println(err)
        return nil, err
    }
     fmt.Println(" After checking blank")
     // Implementing the transaction timestamp feature
    //Transaction id can also be obtained as  stub.UUID
    // Could be leveraged in custom Event listener being planned
    txnTime, err:= stub.GetTxTimestamp()
    if err !=nil {
        err=errors.New("Unable to get transction time")
        return nil, err
    }
    txntimestamp := time.Unix(txnTime.Seconds, int64(txnTime.Nanos))
    sTime := txntimestamp.String()
    if blReg.Timestamp !=nil {
        if strings.TrimSpace(*blReg.Timestamp)=="" {
            blReg.Timestamp = &sTime
        }
    } else {
         blReg.Timestamp = &sTime
    }
    
     fmt.Println(" Timestamp is ", sTime)
    bKey:=*blReg.BLNo                       // Bill of lading record - registration key
   
   // Prepare the record to put Bill of Lading Registration information in the stub
    fmt.Println("First check if it exists in state")
    blStub, err := stub.GetState(bKey)
    if err == nil && (len(blStub)>0) {
        //fmt.Println("blStub ", string(blStub))
        err = errors.New("You cannot create an existing Bill of Lading record: "+ bKey)          
        fmt.Println(err)
		return nil, err
    }

    //If the mandatory fields of Bill of Lading Number, Container Number and Hazmat are satisfactory,
    // and the B/L doesn't already exist in the stub we can put the B/L details into the stub.
    // Set TransitComplete to false
    bTransit := false
    blReg.TransitComplete = &bTransit
    // Marshal back to a JSON string which will be stored in the stub
    regJSON, err := json.Marshal(blReg)
    if err != nil {
        err:=errors.New("Marshaling bill of lading data in registration failed")
        //fmt.Println(err)
        return nil, err
    }
    // Before we put the registration data on the stub, let's create an instance of the state record
    // We won't do an unnecessary stub read for state, since the bill of lading registration doesn't exist
    // If for some unexpected reason it does exist, we don't care -it is invalid and will get overwritten
    
    fmt.Println("before calling container logistics")
    //*************************************************************************
    // Invoke the container logistics contract to create the container's initial state
    
    f := "createContainerLogistics"
    // Fetch container and compliance contract ids from the ledger
    contractStateJson, err := stub.GetState(CONTRACTSTATEKEY)
    if err != nil {
        return nil,errors.New("Unable to fetch container and compliance contract keys")
    }
    err = json.Unmarshal(contractStateJson, &contractState)
    if err != nil {
        return nil, err
    }
    containerChainCode:=contractState.ContainerCC
    complianceChainCode:=contractState.ComplianceCC
    fmt.Println("containerChainCode ", containerChainCode)
    fmt.Println("complianceChainCode ", complianceChainCode)
	var invokeArgs = make([]string, 0) 
    invokeArgs = append(invokeArgs, string(regJSON))
	_, err = stub.InvokeChaincode(containerChainCode, f, invokeArgs)
	if err != nil {
		errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", err.Error())
		fmt.Printf(errStr)
		return nil, errors.New(errStr)
	}
    fmt.Println("before calling compliance")
     //*************************************************************************
    // Invoke the compliance contract to create and maintain the B/L compliance
    f = "createUpdateComplianceRecord"
    compState.BLNo = &bKey
    fmt.Println("compState.BLNo", *compState.BLNo)
    sType := "SHIPPING"
    compState.Type = &sType
    fmt.Println("compState.Type", *compState.Type)
    sTimeStamp:=*blReg.Timestamp 
    compState.Timestamp = &sTimeStamp
    fmt.Println("compState.Timestamp", *compState.Timestamp)
    bCompliance :=true
    compState.Compliance = &bCompliance
    fmt.Println("compState.Compliance", *compState.Compliance)
    bActive := true
    compState.Active=&bActive
    fmt.Println("compState.Active", *compState.Active)
    compJSON, err := json.Marshal(compState)
    if err != nil {
        err:=errors.New("Marshaling compliance initialization failed")
        //fmt.Println(err)
        return nil, err
    }
	fmt.Println("marshal success")
    invokeArgs = make([]string, 0) 
    invokeArgs = append(invokeArgs, string(compJSON))
	_, err = stub.InvokeChaincode(complianceChainCode, f, invokeArgs)
	if err != nil {
		errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", err.Error())
		fmt.Printf(errStr)
		return nil, errors.New(errStr)
	}
     //*************************************************************************
    fmt.Println("After chaincode calls")   
    // Now, lets put the Registration information in the stub

    fmt.Printf("Putting new bill of lading registration data %s to ledger\n", string(regJSON))
    err = stub.PutState(bKey, regJSON)
    if err != nil {
        err:=errors.New("Writing bill of lading registration data to the ledger failed")
        fmt.Println(err)
        return nil, err
    }
	return nil, nil
}

func (t *SimpleChaincode) deregisterBillOfLading(stub *shim.ChaincodeStub, args []string) ([]byte, error) { 
    //call fetchBlData and update the flags to false.
    return nil, nil
    
}

// ************************************
// query functions 
// ************************************

// ***********************getBillOfLadingRegistration************************
    
// ************************************
// getBillOfLadingRegistrationSchema
// ************************************
// This is a 'convenience' function, to provide the consumer of a contract an example of 
// the Bill of Lading definition dataset.
func (t *SimpleChaincode) getBillOfLadingRegistrationSchema(stub *shim.ChaincodeStub, args []string) ([]byte, error) { 
    // Temporarily hardcoded. This is not at present intended to be JSON compatible
    // Can be explanded later to use a combination of ast and default definitions
    bl := []byte (`{ "BLNo": "0000000000", "ContainerNos" : "MSKU000000, MRSK000000",  "Hazmat"  : false,
     "MinTemperature" : -20.00,  "MaxTemperature" : 0.00,   "MinHumidity" : 20.00,  "MaxHumidity" : 50.00,  
     "MinLight" : 0.00,   "MaxLight" : 100.00, "MinAcceleration" : 0.001,  "MaxAcceleration" : 1.9  }`)
      // Notify-aspects are not represented at present. Will be added later
      // Will be replaced by the schema implementation later for consumption by the UI
	return bl, nil
}



// ************************************
// getBillOfLadingRegistration
// ************************************
// This returns the actual Bill of Lading registration dataset.

func (t *SimpleChaincode) getBillOfLadingRegistration(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    var err error
    blRegData, err:=t.fetchBLData( stub, args)
    if err !=nil{
        return nil, err     
    }
    return blRegData, nil
}
// ************************************
// getBillOfLadingState
// ************************************
// This returns the actual Bill of Lading state dataset.


// ************************************
// pullBLData 
// ************************************
// internal utiltiy function

func (t *SimpleChaincode) fetchBLData(stub *shim.ChaincodeStub,  args []string) ([]byte, error) {
    var qKey string
    var err error
    var blReg BillOfLadingRegistration
    if len(args) !=1 {
        err = errors.New("Incorrect number of arguments. Expecting a single JSON string with mandatory BillofLading")
		fmt.Println(err)
		return nil, err
	}
    jsonData:=args[0]
    
    // Marshaling the input to the BillOfLadingRegistration structure. If a value is not
    // sent in, it is defaulted as null
    initJSON := []byte(jsonData)
    fmt.Println("Input asset arg: ", jsonData)
    
    err = json.Unmarshal(initJSON, &blReg)
    if err != nil {
        //err = errors.New("Unable to unmarshal input JSON data")
		fmt.Println(err)
		return nil, err
    }
   // fmt.Println(" blDef after unmarshaling [", blDef, "]") 
   
   //  Nil check for mandatory fields. Since these are defined as pointers in the
   // struct definition, they will be unmarshalled as null (json) / nil (golang)      
     if blReg.BLNo==nil{
        err = errors.New("Bill of Lading is mandatory")
        fmt.Println(err)
		return nil, err
     }
   //  fmt.Println(" Trimming blanks out")
   
   // Check if Bill of Lading and container numbers have been sent in correctly
     *blReg.BLNo = strings.TrimSpace(*blReg.BLNo)
     
     if *blReg.BLNo=="" {
        err = errors.New("Bill of Lading cannot be blank")
        fmt.Println(err)
        return nil, err
    }
    
   
    qKey=*blReg.BLNo                       // Bill of lading record - registration 
    
    blData, err := stub.GetState(qKey)
    if err!=nil {
        err:=errors.New("Unable to retrieve Bill of Lading data from the stub")
        fmt.Println(err)
        return nil, err
    }
    // if it was a reg call, it will be reg info
    // if state call, state info is returned
    fmt.Println(string(blData)) 
    return blData, nil
}
