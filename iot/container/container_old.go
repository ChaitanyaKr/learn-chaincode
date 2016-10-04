


package main
import (
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "reflect"
 //  "github.com/mcuadros/go-jsonschema-generator"
"github.com/hyperledger/fabric/core/chaincode/shim"
)


const CONTRACTSTATEKEY string = "CONTSTATEKEY" 

const BLSTATE string = "_STATE"
// const CONTHIST string = "_HIST"

type ContractState struct {
    Version         string                        `json:"version"`
    ComplianceCC    string                        `json:"compliancecc"`
}   
//const DEFAULT_SCHEMA = "http://json-schema.org/schema#"

const MYVERSION string = "1.0.0"

type variation string

const (
    normal variation ="normal"
    above ="above"
    below = "below" 
)   

// These are common alerts reported by sensor. Example Tetis. 
// http://www.starcomsystems.com/download/Tetis_ENG.pdf 

type Alerts struct {
     TempAlert      *variation `json:"tempalert,omitempty"`
     HumAlert       *variation `json:"humalert,omitempty"`
     LightAlert     *variation `json:"lightalert,omitempty"` 
     AccAlert       *variation `json:"accalert,omitempty"`
     DoorAlert      *bool      `json:"dooralert,omitempty"`
}
// This is  optional. It stands for the 'acceptable range', say 1 degree of lat and long
// at which the container should be, before it is considered 'arrived' at 'Notified Party' location'
// If not sent in, some default could be assumed (say 1 degree or )
type NotifyRange struct {
    LatRange        float64 `json:"latrange,omitempty"`
    LongRange       float64 `json:"longrange,omitempty"`
}
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
// This is a logistics contract, written in the context of shipping. It tracks the progress of a Bill of Lading 
// and associated containers, and raises alerts in case of violations in expected conditions

// Assumption 1. Bill of Lading is sacrosanct - Freight forwarders may issue intermediary freight bills, but
// the original B/L is the document we trackend to end. Similarly a 'Corrected B/L' scenario is not considered

// Assumption 2. A Bill of Lading can have multiple containers attached to it. We are, for simplicity, assuming that
// the same transit rules in terms of allowed ranges in temperature, humidity etc. apply across the B/L - i.e. 
// applies to all containers attached to a Bill of Lading

// Assumption 3. A shipment may switch from one container to another in transit, for various reasons. We are assuming,
// for simplicity, that the same containers are used for end to end transit.
type Geolocation struct {
    Latitude    float64 `json:"latitude,omitempty"`
    Longitude   float64 `json:"longitude,omitempty"`
}

//Structure for logistics data at the container level
type ContainerLogistics struct {
    ContainerNo         *string                         `json:"containerno"`    
    BLNo                *string                         `json:"blno,omitempty"`    
    Location            *Geolocation                    `json:"location,omitempty"`       // current asset location
    Carrier             *string                         `json:"carrier,omitempty"`        // the name of the carrier
    Timestamp           *string                         `json:"timestamp"`          
    Temperature         *float64                        `json:"temperature,omitempty"`    // celcius
    Humidity            *float64                        `json:"humidity,omitempty"` // percent
    Light               *float64                        `json:"light,omitempty"` // lumen
    Acceleration        *float64                        `json:"acceleration,omitempty"`
    DoorClosed          *bool                           `json:"doorclosed,omitempty"`
    Extra               *json.RawMessage                `json:"extra,omitempty"`  
    AlertRecord         *string                         `json:"alerts,omitempty"`  
    TransitComplete     *bool                           `json:"transitcomplete,omitempty"`
}


// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
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
//Container History
type ContainerHistory struct {
	ContHistory []string `json:"conthistory"`
}
var contractState = ContractState{MYVERSION, ""}
var blDefn BillOfLadingRegistration

//************* main *******************
//Create SimpleChaincode instance
//************* main *******************

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple Chaincode: %s", err)
	}
}


// ************************************
// invoke callback mode 
// ************************************
func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	if function == "createContainerLogistics" {
		// create container records
		return t.createContainerLogistics(stub, args)
    } else if function =="updateContainerLogistics" {
        return t.updateContainerLogistics(stub, args)
    } 
	fmt.Println("Unknown invocation function: ", function)
	return nil, errors.New("Received unknown invocation: " + function)
}

// ************************************
// query callback mode 
// ************************************
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	// Handle different Query functions 
	if function =="readContainerLogisitcsSchema" {
        return t.readContainerLogisitcsSchema(stub, args)
    } else if function =="readContainerCurrentStatus" {
        return t.readContainerCurrentStatus(stub, args)
    } else if function =="readContainerHistory" {
            return t.readContainerHistory(stub, args)
    }
	return nil, errors.New("Received unknown invocation: " + function)
}                   
// ************************************
// deploy functions 
// ************************************

//************* init *******************
//Chaincode initialization
//************* init *******************

func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub,  function string, args []string) ([]byte, error) {
	var stateArg ContractState
	var err error

    if len(args) != 1 {
        return nil, errors.New("init expects one argument, a JSON string with tagged version string and chaincode uuid for the compliance")
    }
    err = json.Unmarshal([]byte(args[0]), &stateArg)
    if err != nil {
        return nil, errors.New("Version argument unmarshal failed: " + fmt.Sprint(err))
    }
    if stateArg.Version != MYVERSION {
        return nil, errors.New("Contract version " + MYVERSION + " must match version argument: " + stateArg.Version)
    }
    // set the chaincode uuid of the compliance contract 
    // to the global variable
    
    if stateArg.ComplianceCC =="" {
        return nil, errors.New("Compliance chaincode id is mandatory")
    }
    
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
// invoke functions 
// ************************************

func (t *SimpleChaincode) createContainerLogistics(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    var contInit BillOfLadingRegistration
	var err error
    var contState, oldContState ContainerLogistics
    //var contHistory ContainerHistory
    if len(args) != 1 {
        return nil, errors.New("Expects one argument, a JSON string with bill of lading and container details")
    }
    err = json.Unmarshal([]byte(args[0]), &contInit)
    if err != nil {
        return nil, errors.New("Unable to unmarshal container init " + fmt.Sprint(err))
    }
    // Put the bill of lading data in the global variable, so that the container's
    // validation again B/L rules can be easily accomplished
    blDefn = contInit
    fmt.Println("Splitting the container list")
    bKey:=*contInit.BLNo
    sContainers:=strings.Split(*contInit.ContainerNos, ",")
    sTimeStamp:=*contInit.Timestamp
    iNos:=len(sContainers)
    mTemp := make(map[string]string, iNos)
    for i := 0; i < len(sContainers); i++ {
        fmt.Println("Inside container list iteration")
        bCreateContRecord := true // create a new container record
        //This creates a map of [container number] [alerts] with compliance as a true 
        sContKey:=sContainers[i]
	    mTemp[sContKey]= "" // initializing alerts as a blank string
        // Create an initial ContainerLogisitcs record for each container, in the stub
        // Use the container number for the records. If there is an exisitng record,
        //  update it by appending the B/L. 
        sOldContKey:=sContainers[i] // This will be appended with B/L number for old records
        fmt.Println("Check if old container record exists with the key ", sContKey)
        contData, err := stub.GetState(sContKey)
        if err ==nil && len(contData) >0 {
            fmt.Println(" This container exists in state, probably used with another B/L")
            // If the container number and B/L number match - unlikely, leave untouched
            err = json.Unmarshal(contData, &oldContState)
            if err != nil {
                //err = errors.New("Unable to unmarshal input JSON data")
                fmt.Println(err)
                return nil, err
            }
            if *oldContState.BLNo != bKey {
                // The container - bill of lading combination does not exist
                // this is the expected case
                sOldContKey = sOldContKey + "_" + *oldContState.BLNo 
                // We are going to append the old container record's bill of lading number to the contaienr number
                // This will be the new key for the old record
                 err = stub.PutState(sOldContKey, contData)
                if err != nil {
                    err:=errors.New("re-assigning old container state failed")
                    fmt.Println(err)
                    return nil, err
                }
               // here bCreateContRecord is true 
            } else {
                // If the bill of lading number is same - shouldnt be - do nothing 
                bCreateContRecord = false   
                // Probably throw an error instead...             
            }
        }
        // If there is no data in the stub for the container, or if there was and we reassigned it,
        // we can now create the container's initial record
        // This is needed, because the container record does not come in with a Bill of Lading.
        // Therefore, we map it here 
        if bCreateContRecord {
            fmt.Println("Inside 'bCreateContRecord'")
            contState.ContainerNo=&sContKey
            contState.BLNo=&bKey
            contState.Timestamp = &sTimeStamp
            bTransitComplete:=false
            contState.TransitComplete = &bTransitComplete
            fmt.Println("Before constatate marshal")
            contJSON, err := json.Marshal(contState)
            fmt.Println("After constatate marshal")
            if err != nil {
                err:=errors.New("Marshaling initial container data failed")
                //fmt.Println(err)
                return nil, err
            }
            fmt.Println("Old container record with different B/L", string(contJSON))
            contHistKey:=*contState.ContainerNo+"_HISTORY"
            var contHist = ContainerHistory{make([]string, 1)}
            contHist.ContHistory[0] = string(contJSON)
            contHState, err := json.Marshal(&contHist)
            if err != nil {
                return nil,err
            }
            contHistoryState := []byte(contHState)
            
            err=stub.PutState(sContKey, contJSON)
            if err !=nil {
                fmt.Println("Unable to create initial container state, ", contJSON)
                return nil, err
            }
            err = stub.PutState(contHistKey, contHistoryState)
            if err != nil {
                return nil, errors.New("container history failed PUT to ledger: " + fmt.Sprint(err))
            } 
            fmt.Println("New container record generated", string(contJSON))
            
        }        
    }
    return nil, nil 
} 
 
 /************************ updateContainerLogistics ********************/

func (t *SimpleChaincode) updateContainerLogistics(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var err error
    var contState, contIn   ContainerLogistics
    var compState ComplianceState
    var containerHistory ContainerHistory
    var contractState ContractState
    //var oldAlert, newAlert  Alerts
    
    
    // This invoke function is the heart of the logistics contract.
    // The container details come in one by one and are tagged to the bill of lading.
    // They are validated to ensure no violation have happened. 
    //If yes, invoke the shipping contract for notiifcation - on hold now - only the state gets updated today
     if len(args) !=1 {
        err = errors.New("Incorrect number of arguments. Expecting a single JSON string with mandatory Container Number and optional details")
		fmt.Println(err)
		return nil, err
	}
	jsonData:=args[0]
    conJSON := []byte(jsonData)
    fmt.Println("Input Container data arg: ", jsonData)
    
    // Unmarshal imput data into ContainerLogistics struct   
    err = json.Unmarshal(conJSON, &contIn)
    if err != nil {
        //err = errors.New("Unable to unmarshal input JSON data")
		fmt.Println(err)
		return nil, err
    }
    fmt.Println(" contIn after unmarshaling [", contIn, "]")        
     if contIn.ContainerNo==nil{
         fmt.Println(" Container number is nil")
        err = errors.New("Container number is mandatory")
        fmt.Println(err)
		return nil, err
     }
     
     // Container can't be an empty string
     *contIn.ContainerNo = strings.TrimSpace(*contIn.ContainerNo)
     
     if *contIn.ContainerNo=="" {
        err = errors.New("Container number cannot be blank")
        fmt.Println(err)
        return nil, err
    }
    fmt.Println("After container number check")
    // Check if an initial definiton has been created for this container number
    // If not, an update shouldn't be allowed since we don't know which B/L its associated with.
    // and what are the parameters under which the shipment is supposed to operate
    // Fetch the record in the stub with the container number
    sContKey := *contIn.ContainerNo
    contData, err := stub.GetState(sContKey)
    if err!=nil {
         err = errors.New("Container record not created during registration")
            fmt.Println(err)
            return nil, err
    }
    fmt.Println("After cont state check")
    // This container record has been created in the registration phase
    // or this is not the first container record coming in. 
    fmt.Println("contData ", string(contData))
    err = json.Unmarshal(contData, &contState)
    if err != nil {
        err = errors.New("Unable to unmarshal JSON data from stub")
        fmt.Println(err)
        return nil, err
    }
    
    // Update stub record with new data from the record just read
    // This will be maintained as 'Current State'. This is done because a record may come in 
    // with a partial update. The 'current state' record should be as complete as possible
    // If a property is null in the new record, ignore it, else update the 'state' record
    // using reflection to parse the records and update
    contState, err =t.mergePartialState(contState,contIn)
  
    fmt.Println("After reflection")
    // Container state never comes in with Bill of Lading number. This gets added here
    // The state record already has it, now we add it to the incoming record
    // This is used in the alertsCheck call
    fmt.Println("B/L number in container state is ", *contState.BLNo)
    blKey :=*contState.BLNo
    contIn.BLNo = &blKey
    fmt.Println("B/L number in container in is ", *contIn.BLNo)

    
    fmt.Println("Perform a compliance check on the new record")
    newAlerts, err:= t.alertsCheck(stub, contIn)
    sAlerts := string(newAlerts)
    fmt.Println("Alerts data is : ", string(newAlerts))
    if len(sAlerts)>0 {
        // This implies a compliance violation. 
        fmt.Println("Update container record with new alert status pertaining to that record")
        
        contState.AlertRecord = &sAlerts
        
        // call the compliance contract to maintain the state
        //*************************************************************************
    // Invoke the compliance contract to create and maintain the B/L compliance
        //get compliance contract uuid from the stub
        contractStateJSON, err := stub.GetState(CONTRACTSTATEKEY)
        if err != nil {
            return nil,errors.New("Unable to fetch container and compliance contract keys")
        }
        err = json.Unmarshal(contractStateJSON, &contractState)
        if err != nil {
            return nil, err
        }
        complianceChainCode:=contractState.ComplianceCC
        f := "createUpdateComplianceRecord"
        compKey:=blKey
        compState.BLNo = &compKey
        sType := "SHIPPING"
        compState.Type = &sType
        sTimeStamp:=*contState.Timestamp 
        sCompAlerts:= sAlerts
        mAlerts:= make(map[string]string)
        mAlerts[*contState.ContainerNo]=sCompAlerts
        compState.AssetAlerts=&mAlerts // every alert even triggers a ui action. cumulation not needed. history maintained
        compState.Timestamp = &sTimeStamp
        bCompliance :=false
        compState.Compliance = &bCompliance
        bActive := true
        compState.Active=&bActive
        compJSON, err := json.Marshal(compState)
        if err != nil {
            err:=errors.New("Marshaling compliance initialization failed")
            //fmt.Println(err)
            return nil, err
        }
        invokeArgs := string(compJSON)
        var callArgs = make([]string, 0)
        callArgs = append(callArgs, invokeArgs)
      
        _, err = stub.InvokeChaincode(complianceChainCode, f, callArgs)
        if err != nil {
            errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", err.Error())
            fmt.Printf(errStr)
            return nil, errors.New(errStr)
        }
     //*************************************************************************
       
    }
    
    updContJSON, err := json.Marshal(contState)
    if err != nil {
        err:=errors.New("Marshaling container data failed")
        //fmt.Println(err)
        return nil, err
    }
    
    // Now updated container state data
    fmt.Printf("Putting updated container state data %s to ledger\n", string(updContJSON))
    err = stub.PutState(sContKey, updContJSON)
    if err != nil {
        err:=errors.New("Writing updated container state data to the ledger failed")
        fmt.Println(err)
        return nil, err
    }
    contHistKey:=*contIn.ContainerNo+"_HISTORY"
    var contSlice = make([]string, 0)
    contSlice = append(contSlice, jsonData)
    contSlice = append(contSlice, containerHistory.ContHistory...)
    containerHistory.ContHistory = contSlice
    
    contHState, err := json.Marshal(&containerHistory)
    if err != nil {
        return nil,err
    }
    contHistoryState:= []byte(contHState)
    err = stub.PutState(contHistKey, contHistoryState)
    if err != nil {
        return nil, errors.New("Container history updatefailed PUT to ledger: " + fmt.Sprint(err))
    } 
    
    // Check if lat-long is in notification range
    // Not implementing notification till clarity from IRL Side
    // later, implement it to call the shipping contract
    fmt.Printf("Container %s state successfully written to ledger : %s\n", sContKey, string(updContJSON))
    return nil, nil
 
}


// ************************************
// query functions 
// ************************************


// ************************************
// getContainerLogisitcsSchema
// ************************************
// This is a 'convenience' function, to provide the consumer of a contract an example of 
// the Bill of Lading definition dataset.
func (t *SimpleChaincode) readContainerLogisitcsSchema(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    cont := []byte(`{ "ContainerNo":"MSKU000000","Location" :{"Latitude":10, "Longitude":15}, "Temperature":2, 
        "Carrier":"Maersk", "Timestamp":"2016-03-03T20:27:23.969676659Z", "Humidity":15, "Light":5,
    "DoorClosed":true, "Acceleration":0}`)
    return cont, nil
}

// ************************************
// getContainerCurrentStatus
// ************************************
// This returns the container data
func (t *SimpleChaincode) readContainerCurrentStatus(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    var err error
    var  contIn   ContainerLogistics
    if len(args) !=1 {
        err = errors.New("Incorrect number of arguments. Expecting a single JSON string with mandatory Container Number")
		fmt.Println(err)
		return nil, err
	}
	jsonData:=args[0]
    conJSON := []byte(jsonData)
    fmt.Println("Input Container data arg: ", jsonData)
    
    // Unmarshal imput data into ContainerLogistics struct   
    err = json.Unmarshal(conJSON, &contIn)
    if err != nil {
        //err = errors.New("Unable to unmarshal input JSON data")
		fmt.Println(err)
		return nil, err
    }
    fmt.Println(" contIn after unmarshaling [", contIn, "]")        
     if contIn.ContainerNo==nil{
         fmt.Println(" Container number is nil")
        err = errors.New("Container number is mandatory")
        fmt.Println(err)
		return nil, err
     }
     
     // Container can't be an empty string
     *contIn.ContainerNo = strings.TrimSpace(*contIn.ContainerNo)
     
     if *contIn.ContainerNo=="" {
        err = errors.New("Container number cannot be blank")
        fmt.Println(err)
        return nil, err
    }
    
    sContKey := *contIn.ContainerNo
    contData, err := stub.GetState(sContKey)
    if err!=nil {
         err = errors.New("Container record not available")
            fmt.Println(err)
            return nil, err
    }
    return contData, nil
}
/*********************************  resetContainerHistory ****************************/
 func (t *SimpleChaincode) readContainerHistory(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
    var  contIn   ContainerLogistics
    //var contHist  ContainerHistory
    var err error
    if len(args) !=1 {
        err = errors.New("Incorrect number of arguments. Expecting a single JSON string with mandatory Container Number")
		fmt.Println(err)
		return nil, err
	}
	jsonData:=args[0]
    conJSON := []byte(jsonData)
    fmt.Println("Input Container data arg: ", jsonData)
    
    // Unmarshal imput data into ContainerLogistics struct   
    err = json.Unmarshal(conJSON, &contIn)
    if err != nil {
        //err = errors.New("Unable to unmarshal input JSON data")
		fmt.Println(err)
		return nil, err
    }
    fmt.Println(" contIn after unmarshaling [", contIn, "]")        
     if contIn.ContainerNo==nil{
         fmt.Println(" Container number is nil")
        err = errors.New("Container number is mandatory")
        fmt.Println(err)
		return nil, err
     }
     
    contHistKey := *contIn.ContainerNo+"_HISTORY"
    conthistory, err := stub.GetState(contHistKey   )
    if err != nil {
        return nil, err
    }
     return conthistory, nil
 }


// ************************************
// alertsCheck
// ************************************
// This is an 'internal' function, to check for alert
func (t *SimpleChaincode) alertsCheck(stub *shim.ChaincodeStub, contIn ContainerLogistics) ([]byte,  error) {
    // I will rework thisd - possibly with reflection
    
    var blReg BillOfLadingRegistration 
    //alerts will not get raised: blReg doesn't exist today in state
    // Expensive?
    
    var alert =new(Alerts)
    var contAlert []byte
    var val variation
    
    bKey:= *contIn.BLNo
    fmt.Println("B/L number  inside alertscheck is ",bKey)
    complianceAlert := false
    //  use value in global variable to check alerts compliance.
    blReg = blDefn
    
    // Temperature alert check
   if (blReg.MinTemperature!=nil || blReg.MaxTemperature!=nil) && (contIn.Temperature!=nil) {
       // There is at least one of temp min or max set in the B/L definition and incoming container data has
       //  a temperature reading. We need to perform a temperature alert check
       tempIn:= *contIn.Temperature
       fmt.Println("Temp in: ", tempIn)
       if blReg.MaxTemperature!=nil {
           fmt.Println("In max temp check")
           if tempIn > *blReg.MaxTemperature {
               fmt.Println("In max temp check 2")
               val =above
               alert.TempAlert = &val
               complianceAlert = true
           }
       } else if blReg.MinTemperature!=nil {
           fmt.Println("In min temp check")
           if tempIn < *blReg.MinTemperature {
               fmt.Println("In min temp check 2")
               val =below
               alert.TempAlert = &val
               complianceAlert = true
           }
       }
   }
   fmt.Println("After temp alert check")
   // Humidity alert check
   if (blReg.MinHumidity!=nil || blReg.MaxHumidity!=nil) && (contIn.Humidity!=nil) {
       // There are humidity min or max set in the B/L definition and incoming container data has
       //  a humidity reading. We need to perform a humidity alert check
       humIn:= *contIn.Humidity
       if blReg.MaxHumidity!=nil {
           if humIn > *blReg.MaxHumidity {
               val = above
               alert.HumAlert = &val
               complianceAlert = true
           }
       } else if blReg.MinHumidity!=nil {
           if humIn < *blReg.MinHumidity {
               val = below
               alert.HumAlert = &val
                complianceAlert = true
           }
       }
   }
   fmt.Println("After humidity alert check")
   // light alert check
   if (blReg.MinLight!=nil || blReg.MaxLight!=nil) && (contIn.Light!=nil) {
       // There are temp min or max set in the B/L definition and incoming container data has
       //  a temperature reading. We need to perform a temperature alert check
       humIn:= *contIn.Light
       if blReg.MaxLight!=nil {
           if humIn > *blReg.MaxLight {
               val = above
               alert.LightAlert = &val
               complianceAlert = true
           }
       } else if blReg.MinLight!=nil {
           if humIn < *blReg.MinLight {
               val = below
               alert.LightAlert = &val
                complianceAlert = true
           }
       }
   }
    fmt.Println("After light alert check")
    // Acceleration alert check
   if (blReg.MinAcceleration!=nil || blReg.MaxAcceleration!=nil) && (contIn.Acceleration!=nil) {
       // There are temp min or max set in the B/L definition and incoming container data has
       //  a temperature reading. We need to perform a temperature alert check
       humIn:= *contIn.Acceleration
       if blReg.MaxAcceleration!=nil {
           if humIn > *blReg.MaxAcceleration {
               val = above
               alert.AccAlert = &val
               complianceAlert = true
           }
       } else if blReg.MinAcceleration!=nil {
           if humIn < *blReg.MinAcceleration {
               val = below
               alert.AccAlert = &val
                complianceAlert = true
           }
       }
   }
    fmt.Println("After acceleration alert check")
   ///////////////////////////////////////////////////
   // Note to self: 
   // Look at Reworking above using reflection or at least a sub-function ?
   ////////////////////////////////////////////
    if contIn.DoorClosed!=nil {
        if *contIn.DoorClosed == false {
            dAlert:=true
            alert.DoorAlert =&dAlert
            complianceAlert = true
        }
        
    }
     fmt.Println("After door alert check")
    if complianceAlert {
        cAlert, err := json.Marshal(&alert)
        if err !=nil {
            err = errors.New("Unable to marshal alert data")
		    fmt.Println(err)
		    return nil, err
        }
        contAlert = cAlert
    } 
      return contAlert, nil
}
/*********************************  internal: mergePartialState ****************************/	
 func (t *SimpleChaincode) mergePartialState(oldState ContainerLogistics, newState ContainerLogistics) (ContainerLogistics,  error) {
     
    old := reflect.ValueOf(&oldState).Elem()
    new := reflect.ValueOf(&newState).Elem()
    for i := 0; i < old.NumField(); i++ {
        oldOne:=old.Field(i)
        newOne:=new.Field(i)
        if ! reflect.ValueOf(newOne.Interface()).IsNil() {
            fmt.Println("New is", newOne.Interface())
            fmt.Println("Old is ",oldOne.Interface())
            oldOne.Set(reflect.Value(newOne))
            fmt.Println("Updated Old is ",oldOne.Interface())
        } else {
            fmt.Println("Old is ",oldOne.Interface())
        }
    }
    return oldState, nil
 }
