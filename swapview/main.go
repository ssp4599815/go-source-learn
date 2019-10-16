package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"time"
)

// 定义一个 进程信息所需要的描述信息
type Info struct {
	Pid  int
	Size int64
	Comm string
}

var (
	// 0x是十六进制的前缀
	// 0x0和0x1就是十六进制的0和1，数值上等于十进制的0和1
	nullBytes  = []byte{0x0} // { } 是赋值
	emptyBytes = []byte(" ") // （ ） 是强制转换
)

func GetInfos() (list []Info) {
	f, err := os.Open("/proc")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	// func (f *File) Readdirnames(n int) (names []string, err error)
	// readdirnames  读取目录f下的所有目录对象，返回一个有n个成员的[]string，切片成员为目录中文件对象的名字，采用目录顺序。
	// 对本函数的下一次调用会返回上一次调用剩余未读取的内容的信息。
	// 如果n>0，Readdir函数会返回一个最多n个成员的切片
	// 如果n<=0，Readdir函数返回目录中剩余所有文件对象的名字构成的切片
	names, err := f.Readdirnames(0)
	if err != nil {
		log.Fatalf("read /proc: %v", err)
	}
	for _, name := range names {
		// strconv.Atoi()  返回字符串表示的整数值，接受正负号。
		pid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		info, err := GetInfo(pid)
		if err != nil || info.Size == 0 {
			continue
		}
		list = append(list, info)
	}
	return
}

// 根据 pid 将 一个 进程的其他信息 录入到 Info 中
func GetInfo(pid int) (info Info, err error) {
	info.Pid = pid
	// type byte = uint8
	var bs []byte // 定义一个 byte 类型的列表
	// 根据 pid  获取 当前进行的 执行命令
	// 在Linux系统中，根据进程号得到进程的命令行参数，常规的做法是读取/proc/{PID}/cmdline

	// func ReadFile(filename string) ([]byte, error)
	// ioutil.ReadFile 从filename指定的文件中读取数据并返回文件的内容。
	// 返回的是一个 字符串序列

	// 注意 ，这里不能使用 :=  因为，bs 和 err 都是一个已经定义好的变量
	// 在多个短变量声明和赋值中，至少有一个新声明的变量出现在左值中
	bs, err = ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return // 如果发生了错误，就直接返回
	}
	// bytes.HasSuffix 类属于 strings 用于判断是否包含一个后缀。
	// 这里用于剔除一个 null 字符
	if bytes.HasSuffix(bs, nullBytes) {
		bs = bs[:len(bs)-1]
	}
	// func Replace(s, old, new []byte, n int) []byte
	// 返回将s中前n个不重叠old切片序列都替换为new的新的切片拷贝，如果n<0会替换所有old子切片。
	info.Comm = string(bytes.Replace(bs, nullBytes, emptyBytes, -1))

	// 根据 pid  获取当前进行的  内存信息
	// 通过分析smaps文件我们可以详细了解进程物理内存的使用情况，
	// 比如mmap文件占用了多少空间、动态内存开辟消耗了多少空间、函数调用栈消耗了多少空间等等。
	bs, err = ioutil.ReadFile(fmt.Sprintf("/proc/%d/smaps", pid))
	if err != nil {
		return
	}

	var total int64 // 用来统计所有的内存信息
	// 根据 换行进行分割
	for _, line := range bytes.Split(bs, []byte("\n")) {
		// 如果有 Swap 前缀
		if bytes.HasPrefix(line, []byte("Swap:")) {
			// func IndexAny(s []byte, chars string) int
			// 字符串chars中的任一utf-8编码在s中第一次出现的位置，如不存在或者chars为空字符串则返回-1
			start := bytes.IndexAny(line, "0123456789")   // 从匹配到数字开始
			end := bytes.Index(line[start:], []byte(" ")) // 到匹配到空格符结束
			// func ParseInt(s string, base int, bitSize int) (i int64, err error)
			// 返回字符串表示的整数值，接受正负号。
			// base指定进制（2到36），如果base为0，则会从字符串前置判断，"0x"是16进制，"0"是8进制，否则是10进制；
			// bitSize指定结果必须能无溢出赋值的整数类型，0、8、16、32、64 分别代表 int、int8、int16、int32、int64；
			// 返回的err是*NumErr类型的，如果语法有误，err.Error = ErrSyntax；如果结果超出类型范围err.Error = ErrRange。
			size, err := strconv.ParseInt(string(line[start:start+end]), 10, 0)
			if err != nil {
				continue // 发生错误的话 ，跳过这一行
			}
			total += size
		}
	}
	info.Size = total * 1024 // 默认为字节，转化问 kb 的形式进行存储
	return
}

var units = []string{"", "K", "M", "G", "T"}

// 根据传入的 size 进行不同的显示方式
func FormatSize(s int64) string {
	unit := 0
	f := float64(s)
	// for 的一种写法
	// 如果 size 大于 1024 就进一位
	for unit < len(units) && f > 1100.0 {
		f /= 1024.0
		unit++  // 相当于 进位
	}
	if unit == 0 {
		return fmt.Sprintf("%dB", int64(f))
	} else {
		return fmt.Sprintf("%.1f%siB", f, units[unit])
	}

}

func main() {
	// 计算一个程序耗时的一个 小技巧
	t0 := time.Now()
	defer func() {
		fmt.Printf("%v\n", time.Now().Sub(t0))
	}()

	slist := GetInfos()
	sort.Slice(slist, func(i, j int) bool {
		return slist[i].Size < slist[j].Size
	})
	fmt.Printf("%5s %9s %s \n", "PID", "SWAP", "COMMAND")
	var total int64
	for _, v := range slist {
		fmt.Printf("%5d %9d %d\n", v.Pid, FormatSize(v.Size), v.Comm)
		total += v.Size
	}
	fmt.Printf("Total: %8s\n", FormatSize(total))

}
