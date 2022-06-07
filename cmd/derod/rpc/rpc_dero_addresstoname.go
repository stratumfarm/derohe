package rpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/dvm"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/graviton"
)

func AddressToName(ctx context.Context, p rpc.AddressToName_Params) (result rpc.AddressToName_Result, err error) {

	defer func() { // safety so if anything wrong happens, we return error
		if r := recover(); r != nil {
			err = fmt.Errorf("panic occured. stack trace %s", debug.Stack())
		}
	}()

	var getsc_result rpc.GetSC_Result
	getsc_result.VariableStringKeys = map[string]interface{}{}
	getsc_result.VariableUint64Keys = map[uint64]interface{}{}
	getsc_result.Balances = map[string]uint64{}

	scid := crypto.HashHexToHash("0000000000000000000000000000000000000000000000000000000000000001")

	topoheight := chain.Load_TOPO_HEIGHT()

	if p.TopoHeight >= 1 {
		topoheight = p.TopoHeight
	}

	toporecord, err := chain.Store.Topo_store.Read(topoheight)
	// we must now fill in compressed ring members
	if err == nil {
		var ss *graviton.Snapshot
		ss, err = chain.Store.Balance_store.LoadSnapshot(toporecord.State_Version)
		if err == nil {
			var sc_data_tree *graviton.Tree
			sc_data_tree, err = ss.GetTree(string(scid[:]))
			if err == nil {
				var zerohash crypto.Hash
				if balance_bytes, err := sc_data_tree.Get(zerohash[:]); err == nil {
					if len(balance_bytes) == 8 {
						getsc_result.Balance = binary.BigEndian.Uint64(balance_bytes[:])
					}
				}
				// user requested all variables
				cursor := sc_data_tree.Cursor()
				var k, v []byte
				for k, v, err = cursor.First(); err == nil; k, v, err = cursor.Next() {
					var vark, varv dvm.Variable

					_ = vark
					_ = varv
					_ = k
					_ = v

					//fmt.Printf("key '%x'  value '%x'\n", k, v)
					if len(k) == 32 && len(v) == 8 { // it's SC balance
						getsc_result.Balances[fmt.Sprintf("%x", k)] = binary.BigEndian.Uint64(v)
					} else if k[len(k)-1] >= 0x3 && k[len(k)-1] < 0x80 && nil == vark.UnmarshalBinary(k) && nil == varv.UnmarshalBinary(v) {
						switch vark.Type {
						case dvm.String:
							if varv.Type == dvm.Uint64 {
								err = fmt.Errorf("UNKNOWN Data type")
								return
							} else {
								getsc_result.VariableStringKeys[vark.ValueString] = fmt.Sprintf("%x", []byte(varv.ValueString))
							}
						default:
							err = fmt.Errorf("UNKNOWN Data type")
							return
						}

					}
				}
			}

		}

	}

	var req_addr *rpc.Address
	if req_addr, err = rpc.NewAddress(strings.TrimSpace(p.Address)); err != nil {
		err = fmt.Errorf("Invalid Address")
		return
	}

	for k, v := range getsc_result.VariableStringKeys {
		b, _ := hex.DecodeString(v.(string))
		addr, err := rpc.NewAddressFromCompressedKeys(b)
		if err != nil {
			//fmt.Printf("%s, %s\n", k, v)
			continue
		}
		if bytes.Equal(req_addr.Compressed(), addr.Compressed()) {
			//fmt.Printf("%s, %s\n", k, v)
			result.Names = append(result.Names, k)
		}
	}

	if len(result.Names) != 0 {
		sort.Strings(result.Names)
		result.Address = p.Address
		result.Status = "OK"
	} else {
		err = fmt.Errorf("Not Found")
		return
	}

	return
}
