package rest

import (
	"errors"
	"fmt"
	"github.com/panyam/relay/bindings"
	"io"
	"os"
	"text/template"
)

/**
 * Responsible for generating the code for the client classes.
 */
type Generator struct {
	// where the templates are
	Bindings     map[string]*HttpBinding
	TypeSystem   bindings.ITypeSystem
	TemplatesDir string

	// Parameters to determine Generated output
	Package           string
	ClientPackageName string
	ServiceName       string
	ClientPrefix      string
	ClientSuffix      string
	httpBindings      map[string]*HttpBinding
	ArgListMaker      func([]*bindings.Type, bool) string
	ServiceType       *bindings.RecordTypeData
	TransportRequest  string
	OpName            string
	OpType            *bindings.FunctionTypeData
	OpMethod          string
	OpEndpoint        string
}

func ArgListMaker(paramTypes []*bindings.Type, withNames bool) string {
	out := ""
	for index, param := range paramTypes {
		if index > 0 {
			out += ", "
		}
		if withNames {
			out += fmt.Sprintf("arg%d ", index)
		}
		out += param.Signature()
	}
	return out
}

func (g *Generator) ClientName() string {
	return g.ClientPrefix + g.ServiceName + g.ClientSuffix
}

func NewGenerator(bindings map[string]*HttpBinding, typeSys bindings.ITypeSystem, templatesDir string) *Generator {
	if bindings == nil {
		bindings = make(map[string]*HttpBinding)
	}
	out := Generator{Bindings: bindings,
		TypeSystem:        typeSys,
		TemplatesDir:      templatesDir,
		ClientPackageName: "restclient",
		ClientSuffix:      "Client",
		TransportRequest:  "*http.Request",
		ArgListMaker:      ArgListMaker,
	}
	return &out
}

/**
 * Emits the class that acts as a client for the service.
 */
func (g *Generator) EmitClientClass(pkgName string, serviceName string) error {
	g.ServiceName = serviceName
	g.ServiceType = g.TypeSystem.GetType(pkgName, serviceName).TypeData.(*bindings.RecordTypeData)

	tmpl, err := template.New("client.gen").ParseFiles(g.TemplatesDir + "client.gen")
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, g)
	if err != nil {
		panic(err)
	}
	return err
}

/**
 * For a given service operation, emits a method which:
 * 1. Has inputs the same as those of the underlying service operation,
 * 2. creates a transport level request
 * 3. Sends the transport level request
 * 4. Gets a response from the transport level and returns it
 */
func (g *Generator) EmitSendRequestMethod(output io.Writer, opName string, opType *bindings.FunctionTypeData, argPrefix string) error {
	g.OpName = opName
	g.OpType = opType
	g.OpMethod = "GET"
	g.OpEndpoint = "http://hello.world/"
	g.StartWritingMethod(output, opName, opType, "arg")
	if opType.NumInputs() > 0 {
		if opType.NumInputs() == 1 {
			g.EmitObjectWriterCall(output, nil, "arg0", opType.InputTypes[0])
		} else {
			g.StartWritingList(output)
			for index, param := range opType.InputTypes {
				g.EmitObjectWriterCall(output, index, fmt.Sprintf("arg%d", index), param)
			}
			g.EndWritingList(output)
		}
	}
	g.EndWritingMethod(output, opName, opType)
	return nil
}

func (g *Generator) StartWritingMethod(output io.Writer, opName string, opType *bindings.FunctionTypeData, argPrefix string) error {
	templ, err := template.New("writer").Parse(`
func (svc *{{$.ClientName}}) Send{{.OpName}}Request({{call .ArgListMaker .OpType.InputTypes true }}) (resp *http.Response, error) {
	body := bytes.NewBuffer(nil)
	`)
	if err != nil {
		panic(err)
	}
	err = templ.Execute(output, g)
	if err != nil {
		panic(err)
	}
	return err
}

func (g *Generator) EndWritingMethod(output io.Writer, opName string, opType *bindings.FunctionTypeData) error {
	templ, err := template.New("writer").Parse(`
	httpreq, err := http.NewRequest("{{.OpMethod}}", "{{.OpEndpoint}}", body)
	if err != nil {
		return nil, err
	}
	httpreq.Header.Add("Content-Type", "application/json")
	if svc.RequestDecorator != nil {
		httpreq, err = svc.RequestDecorator(httpreq)
		if err != nil { return nil, err }
	}
	c := http.Client{}
	return c.Do(httpreq)
}
	`)
	if err != nil {
		panic(err)
	}
	err = templ.Execute(output, g)
	if err != nil {
		panic(err)
	}
	return err
}

func WriterMethodForType(t *bindings.Type) string {
	switch typeData := t.TypeData.(type) {
	case string:
		return "Write_" + typeData
	case *bindings.AliasTypeData:
		return WriterMethodForType(typeData.AliasFor)
	case *bindings.ReferenceTypeData:
		return WriterMethodForType(typeData.TargetType)
	case *bindings.FunctionTypeData:
		panic(errors.New("Function types not supported in GO"))
	case *bindings.TupleTypeData:
		panic(errors.New("Warning: Tuple types not supported in GO"))
		return "Write_Tuple"
	case *bindings.RecordTypeData:
		return "Write_" + typeData.Name
	case *bindings.MapTypeData:
		return "Write_Map"
	case *bindings.ListTypeData:
		return "Write_List"
	}
	return "UnknownWriter"
}

/**
 * Emits the code required to invoke the serializer of an object of a given
 * type.
 */
func (g *Generator) EmitObjectWriterCall(output io.Writer, key interface{}, argName string, argType *bindings.Type) error {
	callString := WriterMethodForType(argType)
	output.Write([]byte(callString + "(body, " + argName + ")"))
	return nil
}

/**
 * Emits the code required to start a list.
 */
func (g *Generator) StartWritingList(output io.Writer) {
	output.Write([]byte("["))
}

/**
 * Emits the code required to end a list.
 */
func (g *Generator) EndWritingList(output io.Writer) {
	output.Write([]byte("]"))
}

/**
 * For a given service operation, emits a method:
 * 1. whose input is a http.Response object
 * 2. Which can be parsed into the output values as expected by the service
 * 	  operations's output signature
 */
/*
func (g *Generator) EmitReadResponseMethod(opName string, opType *bindings.FunctionTypeData, argPrefix string) error {
	g.StartReadingMethod(opName, opType, "arg")
	if opType.NumOutputs() > 0 {
		if opType.NumOutputs() == 1 {
			g.EmitObjectReaderCall("arg0", opType.OutputTypes[0])
		} else {
			g.StartReadingList()
			for index, param := range opType.OutputTypes {
				g.StartReadingChild()
				g.EmitObjectReaderCall(fmt.Sprintf("arg%d", index), param)
				g.EndReadingChild()
			}
			g.EndReadingList()
		}
	}
	g.EndReadingMethod(opName, opType)
}
*/
