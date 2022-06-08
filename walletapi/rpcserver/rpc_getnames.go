package rpcserver

import "fmt"
import "context"
import "runtime/debug"

//import	"log"
//import 	"net/http"

import "github.com/deroproject/derohe/rpc"

func GetNames(ctx context.Context) (result rpc.GetNames_Result, err error) {
	defer func() { // safety so if anything wrong happens, we return error
		if r := recover(); r != nil {
			err = fmt.Errorf("panic occured. stack trace %s", debug.Stack())
		}
	}()
	w := fromContext(ctx)
	addr := w.wallet.GetAddress().String()
	names, err := w.wallet.AddressToName(addr)
	return rpc.GetNames_Result{
		Names: names,
	}, err
}
