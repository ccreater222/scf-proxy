package scf

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
)

var (
	ScfApiProxyUrl string
)
func unitRot13(x byte) byte {
	capital := x >= 'A' && x <= 'Z'
	if !capital && (x < 'a' || x > 'z') {
		return x // Not a letter
	}

	x += 13
	if capital && x > 'Z' || !capital && x > 'z' {
		x -= 26
	}
	return x
}
func Rot13(x []byte)  []byte {
	b := bytes.NewBuffer([]byte{})
	for i := range x{
		b.WriteByte(unitRot13(x[i]))
	}
	return b.Bytes()
}
func ParseHeader(x string)(result map[string]string){
	x = strings.Trim(x,"\r\n ")
	x = strings.Replace(x,"\r\n","\n",-1)
	paresd := strings.Split(x,"\n")
	result = make(map[string]string,len(paresd))
	for i:=0;i<len(paresd);i+=1{
		kvmap := strings.SplitN(paresd[i],": ",2)
		if len(kvmap)!=2{
			continue
		}
		result[kvmap[0]] = kvmap[1]
	}
	return result
}

func HandlerHttp(w http.ResponseWriter, r *http.Request) {
	dumpReq, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event := &DefineEvent{
		URL:     r.URL.String(),
		Content: string(Rot13([]byte(base64.StdEncoding.EncodeToString(dumpReq)))),
	}
	bytejson, err := json.Marshal(event)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	req, err := http.NewRequest("POST", ScfApiProxyUrl, bytes.NewReader(bytejson))
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("client.Do()", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	bytersp, _ := ioutil.ReadAll(resp.Body)

	var respevent RespEvent
	if err := json.Unmarshal(bytersp, &respevent); err != nil {
		log.Println(err)
		log.Println(string(bytersp))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if resp.StatusCode > 0 && resp.StatusCode != 200 {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	retByte, err := base64.StdEncoding.DecodeString(string(Rot13(bytes.NewBufferString(respevent.Data).Bytes())))
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	headers,err := base64.StdEncoding.DecodeString(string(Rot13(bytes.NewBufferString(respevent.Header).Bytes())))
	if err != nil {
		log.Println(err)
		log.Println(string(respevent.Data))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	header_map :=ParseHeader(string(headers))
	for header,value := range header_map{
		if w.Header().Get(header)==""{
			w.Header().Add(header,value)
		}
	}


	resp.Body.Close()

	w.Write(retByte)
	return
}
