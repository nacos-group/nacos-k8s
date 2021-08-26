package errors

import (
	"encoding/json"
	"fmt"
)

type Err struct {
	Code int
	Msg  string
}

func (e *Err) Error() string {
	err, _ := json.Marshal(e)
	return string(err)
}

func EnsureNormal(err error) {
	if err != nil {
		panic(err)
	}
}

func EnsureNormalMyError(err error, code int) {
	if err != nil {
		panic(New(code, err.Error()))
	}
}

func EnsureEqual(i1 interface{}, i2 interface{}, code int, msg ...interface{}) {
	if i1 != i2 {
		panic(New(code, "msg: %v, %v is not equal %v", msg, i1, i2))
	}
}

func EnsureNormalMsgf(err error, format string, a ...interface{}) {
	if err != nil {
		panic(err)
	}
}

func New(code int, format string, a ...interface{}) *Err {
	msg := fmt.Sprintf(format, a...)
	if len(a) == 0 {
		msg = fmt.Sprint(format, a)
	}
	return &Err{
		Code: code,
		Msg:  msg,
	}
}

func NewErr(err error) *Err {
	return &Err{
		Code: CODE_ERR_UNKNOW,
		Msg:  err.Error(),
	}
}
func NewErrMsg(err string) *Err {
	return &Err{
		Code: CODE_ERR_UNKNOW,
		Msg:  err,
	}
}

func NewErrfMsgf(format string, a ...interface{}) *Err {
	msg := fmt.Sprintf(format, a...)
	if len(a) == 0 {
		msg = fmt.Sprint(format, a)
	}
	return &Err{
		Code: CODE_ERR_UNKNOW,
		Msg:  msg,
	}
}
