package qtcp

import "fmt"

// Check 检测并抛出异常
func Check(e error) {
	if e != nil {
		panic(e)
	}
}

// Recover 收集异常
func Recover() error {
	if err := recover(); err != nil {
		fmt.Println(err)
		return err.(error)
	}
	return nil
}
