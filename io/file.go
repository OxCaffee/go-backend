package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
)

// 打开和关闭文件
func OpenFile() {
	filePtr, err := os.Open("./file.dat")
	if err != nil {
		fmt.Println("打开文件失败！错误原因：", err)
	} else {
		fmt.Println("打开文件成功！")
	}

	// 记得关闭文件
	defer func() {
		filePtr.Close()
		fmt.Println("文件关闭成功")
	}()

	fmt.Println("文件打开关闭操作已经结束")
	<-time.After(2 * time.Second)
}

// 读取文件数据
func ReadData01() {
	// 打开和关闭文件
	// os.O_RDONLY以只读方式打开
	// 权限为 主任/用户组/别人
	filePtr, err := os.Open("./file.dat")
	if err != nil {
		fmt.Println("文件打开失败！错误原因：", err)
		return
	} else {
		fmt.Println("文件打开成功!")
	}

	// 记得关闭文件
	defer func() {
		filePtr.Close()
		fmt.Println("文件已经关闭")
	}()

	// 开始读取文件
	reader := bufio.NewReader(filePtr)

	// 循环读取
	for {
		// 以换行符为界，分批次读取数据，得到string
		str, err := reader.ReadString('\n')
		if err != nil {

			// 如果已经到文件末尾
			if err == io.EOF {
				break
			}

			// 如果是其他的错误，显示文件读取失败
			fmt.Println("文件读取失败！")
		}

		// 打印读取到的数据
		fmt.Print(str)
	}

}

// 使用ioutil读取文件数据
func ReadData02() {
	// ioutil内部调用了文件的打开和关闭
	bytes, err := ioutil.ReadFile("./file.dat")
	if err != nil {
		fmt.Println("文件打开失败！错误原因：", err)
		return
	} else {
		fmt.Println("文件打开成功!")
	}

	text := string(bytes)

	fmt.Println(text)
}

func WriteData01() {
	// 打开文件
	// 创建并写入模式
	fp1, err01 := os.OpenFile("./file02.dat", os.O_CREATE|os.O_WRONLY, 0754)
	// 覆盖写模式
	fp2, err02 := os.OpenFile("./file03.dat", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0754)
	// 追加写模式
	fp3, err03 := os.OpenFile("./file04.dat", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0754)

	if err01 != nil || err02 != nil || err03 != nil {
		fmt.Println("文件打开失败！")
		return
	}

	defer fp1.Close()
	defer fp2.Close()
	defer fp3.Close()

	writer01 := bufio.NewWriter(fp1)
	writer02 := bufio.NewWriter(fp2)
	writer03 := bufio.NewWriter(fp3)

	writer01.WriteString("file1")
	writer02.WriteString("file2")
	writer03.WriteString("file3")

	writer01.Flush()
	writer02.Flush()
	writer03.Flush()

	fmt.Println("写入成功~！")
}

func main() {
	//OpenFile()
	// ReadData01()
	// ReadData02()
	WriteData01()
	fmt.Println()
}
