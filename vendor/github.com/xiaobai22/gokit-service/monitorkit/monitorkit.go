package monitorkit

import (
	"github.com/xiaobai22/gokit-service/blackboardkit"
	"github.com/xiaobai22/gokit-service/httpkit"

	"fmt"
	"net/http"
)

func monitor(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if co := recover(); co != nil {

		}
	}()
	//fmt.Println("yes")
	//time.Sleep(time.Second*1 )
	r.ParseForm()
	str := blackboardkit.ALLBB.Show()
	fmt.Fprintf(w, str)
}

func StartMonitorBB(port string,path string  ) {
	go func() {
		defer func() {
			if co := recover(); co != nil {

			}
		}()
		httpkit.NewSimpleHttpServer().Add(path, monitor).Start(port)
	}()
}
