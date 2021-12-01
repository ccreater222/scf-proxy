package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"scf-proxy/pkg/scf"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/tencentyun/scf-go-lib/events"
)

func main() {
	// log.SetOutput(os.Stdout)
	cloudfunction.Start(server)
}
func init() {
	fmt.Println("hello world")
	log.SetFormatter(new(Formatter))
}
func server(context context.Context, request events.APIGatewayRequest) (response events.APIGatewayResponse, err error) {
	bytedata, _ := json.Marshal(request)
	log.Printf("[Receive] \n => context \n%v \n => request \n%s \n========== \n\n", context, string(bytedata))

	return scf.Handler(context, request), nil
}

type Formatter struct{}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	var buffer *bytes.Buffer
	if entry.Buffer != nil {
		buffer = entry.Buffer
	} else {
		buffer = &bytes.Buffer{}
	}

	buffer.WriteString("[")
	buffer.WriteString(entry.Time.Format("2006-01-02 15:04:05.999999999"))
	buffer.WriteString("]")

	buffer.WriteString("[")
	buffer.WriteString(entry.Level.String())
	buffer.WriteString("]")

	if entry.HasCaller() {
		buffer.Write([]byte("["))
		buffer.Write([]byte(fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)))
		buffer.Write([]byte("] "))

		// buffer.Write([]byte("[fn] "))
		// buffer.Write([]byte(entry.Caller.Func.Name()))
		// buffer.WriteString(" ")
	}
	buffer.WriteString(entry.Message)
	buffer.WriteString("\n")
	return buffer.Bytes(), nil
}
