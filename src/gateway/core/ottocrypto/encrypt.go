package ottocrypto

import (
	"encoding/base64"
	"gateway/crypto"
	"gateway/logreport"

	"github.com/robertkrimen/otto"
)

// IncludeEncryption adds the AP.Crypto.encrypt and AP.Crypto.decrypt functions in
// the supplied Otto VM.
func IncludeEncryption(vm *otto.Otto, accountID int64, keySource KeyDataSource) {
	setEncrypt(vm, accountID, keySource)
	setDecrypt(vm, accountID, keySource)

	scripts := []string{
		"AP.Crypto.encrypt = _encrypt; delete _encrypt;",
		"AP.Crypto.decrypt = _decrypt; delete _decrypt;",
	}

	for _, s := range scripts {
		vm.Run(s)
	}
}

func setEncrypt(vm *otto.Otto, accountID int64, keySource KeyDataSource) {
	vm.Set("_encrypt", func(call otto.FunctionCall) otto.Value {
		d, err := getArgument(call, 0)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		var data string
		if ds, ok := d.(string); ok {
			data = ds
		} else {
			logreport.Println("data should be a string")
			return undefined
		}

		o, err := getArgument(call, 1)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		key, algorithm, tag, err := getOptions(o, keySource, accountID)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		result, err := crypto.Encrypt([]byte(data), key, algorithm, tag)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		val, err := vm.ToValue(result)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		return val
	})
}

func setDecrypt(vm *otto.Otto, accountID int64, keySource KeyDataSource) {
	vm.Set("_decrypt", func(call otto.FunctionCall) otto.Value {
		d, err := getArgument(call, 0)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		ds, ok := d.(string)
		if !ok {
			logreport.Println("data should be a string")
			return undefined
		}

		o, err := getArgument(call, 1)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		key, algorithm, tag, err := getOptions(o, keySource, accountID)
		if err != nil {
			logreport.Println(err)
			return undefined
		}

		data, err := base64.StdEncoding.DecodeString(ds)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		result, err := crypto.Decrypt(data, key, algorithm, tag)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		val, err := vm.ToValue(result)
		if err != nil {
			logreport.Print(err)
			return undefined
		}

		return val
	})
}