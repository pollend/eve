package main

import (
	"fmt"
	//"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"io/ioutil"
	"github.com/golang/protobuf/proto"
	"github.com/shirou/gopsutil/net"
	"shared/proto/zmet"
	"zc/libs/zmsg"
	"time"
	"net/http"
	"bytes"
)

var networkStat [][]string
var cpuStorageStat [][]string

const (
	statusURL string = "http://192.168.1.21:8088/api/v1/edgedevice/info"
)
const (
	metricsURL string = "http://192.168.1.21:8088/api/v1/edgedevice/metrics"
)

func publishMetrics() {
	DeviceCpuStorageStat()
	DeviceNetworkStat()
	MakeMetricsProtobufStructure()
	MakeAppInfoProtobufStructure()
	MakeDeviceInfoProtobufStructure()
	MakeHypervisorInfoProtobufStructure()
}

func metricsTimerTask() {
	ticker := time.NewTicker(time.Second  * 5)
	for t := range ticker.C {
		fmt.Println("Tick at", t)
		publishMetrics();
	}
}

func DeviceCpuStorageStat() {
	count := 0
	counter := 0
	app0 := "sudo"
	app := "xentop"
	arg0 := "-b"
	//arg1 := "-n"
	arg4 := "-d"
	arg5 := "1"
	arg2 := "-i"
	arg3 := "2"

	cmd1 := exec.Command(app0, app, arg0, arg4, arg5, arg2, arg3)
	stdout, err := cmd1.Output()
	if err != nil {
		println(err.Error())
		return
	}
	//fmt.Println(string(stdout))

	xentopInfo := fmt.Sprintf("%s", stdout)

	splitXentopInfo := strings.Split(xentopInfo, "\n")

	splitXentopInfoLength := len(splitXentopInfo)
	//fmt.Println("splitXentopInfoLength: ",splitXentopInfoLength)
	var i int
	var start int

	for i = 0; i < splitXentopInfoLength; i++ {

		str := fmt.Sprintf(splitXentopInfo[i])
		re := regexp.MustCompile(" ")

		spaceRemovedsplitXentopInfo := re.ReplaceAllLiteralString(splitXentopInfo[i], "")
		matched, err := regexp.MatchString("NAMESTATECPU.*", spaceRemovedsplitXentopInfo)

		if matched {

			count++
			fmt.Sprintf("string matched: ", str)
			if count == 2 {

				start = i
				fmt.Sprintf("value of i: ", start)
			}

		} else {
			fmt.Sprintf("string not matched", err)
		}
	}

	length := splitXentopInfoLength - 1 - start
	//var finalOutput [length] string
	finalOutput := make([][]string, length)
	//fmt.Println(len(finalOutput))

	for j := start; j < splitXentopInfoLength-1; j++ {

		str := fmt.Sprintf(splitXentopInfo[j])
		splitOutput := regexp.MustCompile(" ")
		finalOutput[j-start] = splitOutput.Split(str, -1)
	}

	//var cpuStorageStat [][]string
	cpuStorageStat = make([][]string, length)

	for i := range cpuStorageStat {
		cpuStorageStat[i] = make([]string, 20)
	}

	for f := 0; f < length; f++ {

		for out := 0; out < len(finalOutput[f]); out++ {

			//fmt.Println(finalOutput[f][out])
			matched, err := regexp.MatchString("[A-Za-z0-9]+", finalOutput[f][out])
			fmt.Sprint(err)
			if matched {

				if finalOutput[f][out] == "no" {

				} else if finalOutput[f][out] == "limit" {
					counter++
					cpuStorageStat[f][counter] = "no limit"
					//fmt.Println("f : out: ",f,counter,cpuStorageStat[f][counter])
				} else {
					counter++
					cpuStorageStat[f][counter] = finalOutput[f][out]
					//fmt.Println("f : out: ",f,counter,cpuStorageStat[f][counter])
				}
			} else {

				fmt.Sprintf("space: ", finalOutput[f][counter])
			}
		}
		counter = 0
	}
}

func DeviceNetworkStat() {

	counter := 0
	netDetails,err := ioutil.ReadFile("/proc/net/dev")
	if err != nil {
		fmt.Println(err)
	}

	networkInfo := fmt.Sprintf("%s", netDetails)
	splitNetworkInfo := strings.Split(networkInfo, "\n")
	splitNetworkInfoLength := len(splitNetworkInfo)
	length := splitNetworkInfoLength - 1

	finalNetworStatOutput := make([][]string, length)

	for j := 0; j < splitNetworkInfoLength-1; j++ {

		str := fmt.Sprintf(splitNetworkInfo[j])
		splitOutput := regexp.MustCompile(" ")
		finalNetworStatOutput[j] = splitOutput.Split(str, -1)
	}

	//var networkStat [][]string
	networkStat = make([][]string, length)

	for i := range networkStat {
		networkStat[i] = make([]string, 20)
	}

	for f := 0; f < length; f++ {

		for out := 0; out < len(finalNetworStatOutput[f]); out++ {

			//fmt.Println(finalNetworStatOutput[f][out])
			matched, err := regexp.MatchString("[A-Za-z0-9]+", finalNetworStatOutput[f][out])
			fmt.Sprint(err)
			if matched {
				counter++
				networkStat[f][counter] = finalNetworStatOutput[f][out]
				//fmt.Println("f : out: ",f,counter,networkStat[f][counter])
			}
		}
		counter = 0
	}
}

func MakeMetricsProtobufStructure() {

	var ReportMetricsToZedCloud = &zmet.ZMetricMsg{}

	ReportDeviceMetric := new(zmet.DeviceMetric)
	ReportDeviceMetric.Cpu		 = new(zmet.CpuMetric)
	ReportDeviceMetric.Memory	 = new(zmet.MemoryMetric)
	ReportDeviceMetric.Network	 = make([]*zmet.NetworkMetric, len(networkStat)-2)

	ReportMetricsToZedCloud.DevID = *proto.String("8f2238e7-948d-4601-a384-644c1b39467a")
	ReportZmetric := new(zmet.ZmetricTypes)
	*ReportZmetric = zmet.ZmetricTypes_ZmDevice

	ReportMetricsToZedCloud.Ztype = *ReportZmetric

	for arr := 1; arr < 2; arr++ {

		cpuTime, _ := strconv.ParseUint(cpuStorageStat[arr][3], 10, 0)
		ReportDeviceMetric.Cpu.UpTime = *proto.Uint32(uint32(cpuTime))

		cpuUsedInPercent, _ := strconv.ParseFloat(cpuStorageStat[arr][4], 10)
		ReportDeviceMetric.Cpu.CpuUtilization = *proto.Float32(float32(cpuUsedInPercent))

		memory, _ := strconv.ParseUint(cpuStorageStat[arr][5], 10, 0)
		ReportDeviceMetric.Memory.UsedMem = *proto.Uint32(uint32(memory))

		memoryUsedInPercent, _ := strconv.ParseFloat(cpuStorageStat[arr][6], 10)
		ReportDeviceMetric.Memory.UsedPercentage = *proto.Float32(float32(memoryUsedInPercent))

		maxMemory, _ := strconv.ParseUint(cpuStorageStat[arr][7], 10, 0)
		ReportDeviceMetric.Memory.MaxMem = *proto.Uint32(uint32(maxMemory))

		for net := 2; net < len(networkStat); net++ {

			//fmt.Println(networkStat[2][1])
			networkDetails := new(zmet.NetworkMetric)
			networkDetails.DevName = *proto.String(networkStat[net][1])

			txBytes, _ := strconv.ParseUint(networkStat[net][10], 10, 0)
			networkDetails.TxBytes = *proto.Uint64(txBytes)
			rxBytes, _ := strconv.ParseUint(networkStat[net][2], 10, 0)
			networkDetails.RxBytes = *proto.Uint64(rxBytes)

			txDrops, _ := strconv.ParseUint(networkStat[net][13], 10, 0)
			networkDetails.TxDrops = *proto.Uint64(txDrops)
			rxDrops, _ := strconv.ParseUint(networkStat[net][5], 10, 0)
			networkDetails.RxDrops = *proto.Uint64(rxDrops)
			// assume rx and tx rates 0 for now...
			txRate, _ := strconv.ParseUint("0", 10, 0)
			networkDetails.TxRate = *proto.Uint64(txRate)
			rxRate, _ := strconv.ParseUint("0", 10, 0)
			networkDetails.RxRate = *proto.Uint64(rxRate)

			ReportDeviceMetric.Network[net-2] = networkDetails
			//fmt.Println(ReportDeviceMetric.Network[net-2])
			ReportMetricsToZedCloud.Dm = ReportDeviceMetric

		}

	}

	//fmt.Printf("%T", ReportMetricsToZedCloud)
	fmt.Println(" ")
	SendMetricsProtobufStrThroughHttp(ReportMetricsToZedCloud)
}
func MakeDeviceInfoProtobufStructure (){

	var ReportInfo = &zmsg.ZInfoMsg{}
	var storage_size = 1

	deviceType			:= new(zmet.ZInfoTypes)
	*deviceType			=	zmet.ZInfoTypes_ZiDevice
	ReportInfo.Ztype	=	*deviceType

	ReportInfo.DevId	=	*proto.String("8f2238e7-948d-4601-a384-644c1b39467a")

	ReportDeviceInfo	:=	new(zmet.ZInfoDevice)
	ReportDeviceInfo.MachineArch	=	*proto.String("32 bit")
	ReportDeviceInfo.CpuArch		=	*proto.String("x86")
	ReportDeviceInfo.Platform		=	*proto.String("ubuntu")
	ReportDeviceInfo.Ncpu			=	*proto.Uint32(uint32(storage_size))
	ReportDeviceInfo.Memory			=	*proto.Uint64(uint64(storage_size))
	ReportDeviceInfo.Storage		=	*proto.Uint64(uint64(storage_size))

	ReportDeviceInfo.Devices	=	make([]*zmet.ZinfoPeripheral,	1)
	ReportDevicePeripheralInfo	:=	new(zmet.ZinfoPeripheral)

	for	index,_	:=	range ReportDeviceInfo.Devices	{

		PeripheralType											:=		new(zmet.ZPeripheralTypes)
		ReportDevicePeripheralManufacturerInfo					:=		new(zmet.ZInfoManufacturer)
		*PeripheralType											=		zmet.ZPeripheralTypes_ZpNone
		ReportDevicePeripheralInfo.Ztype						=		*PeripheralType
		ReportDevicePeripheralInfo.Pluggable					=		*proto.Bool(false)
		ReportDevicePeripheralManufacturerInfo.Manufacturer		=		*proto.String("apple")
		ReportDevicePeripheralManufacturerInfo.ProductName		=		*proto.String("usb")
		ReportDevicePeripheralManufacturerInfo.Version			=		*proto.String("1.2")
		ReportDevicePeripheralManufacturerInfo.SerialNumber		=		*proto.String("1mnah34")
		ReportDevicePeripheralManufacturerInfo.UUID				=		*proto.String("uyapple34")
		ReportDevicePeripheralInfo.Minfo						=		ReportDevicePeripheralManufacturerInfo
		ReportDeviceInfo.Devices[index]							=		ReportDevicePeripheralInfo
	}

	ReportDeviceManufacturerInfo	:=	new(zmet.ZInfoManufacturer)
	ReportDeviceManufacturerInfo.Manufacturer		=		*proto.String("intel")
	ReportDeviceManufacturerInfo.ProductName		=		*proto.String("vbox")
	ReportDeviceManufacturerInfo.Version			=		*proto.String("1.2")
	ReportDeviceManufacturerInfo.SerialNumber		=		*proto.String("acmck11112c")
	ReportDeviceManufacturerInfo.UUID				=		*proto.String("12345")
	ReportDeviceInfo.Minfo							=		ReportDeviceManufacturerInfo

	ReportDeviceSoftwareInfo	:=	new(zmet.ZInfoSW)
	ReportDeviceSoftwareInfo.SwVersion	=		*proto.String("1.1.2")
	ReportDeviceSoftwareInfo.SwHash		=		*proto.String("12awsxlnvme456")
	ReportDeviceInfo.Software			=		ReportDeviceSoftwareInfo

	//find	all	network	related	info...
	interfaces,_	:=	net.Interfaces()
	ReportDeviceInfo.Network	=	make([]*zmet.ZInfoNetwork,	len(interfaces))
	for	index,val	:=	range	interfaces	{

		ReportDeviceNetworkInfo	:=	new(zmet.ZInfoNetwork)
		for	ip := 0;ip < len(val.Addrs) - 1;ip++ {
			ReportDeviceNetworkInfo.IPAddr	=	*proto.String(val.Addrs[0].Addr)
		}

		ReportDeviceNetworkInfo.GwAddr		=	*proto.String("192.168.1.1")
		ReportDeviceNetworkInfo.MacAddr		=	*proto.String(val.HardwareAddr)
		ReportDeviceNetworkInfo.DevName		=	*proto.String(val.Name)
		ReportDeviceInfo.Network[index]		=	ReportDeviceNetworkInfo

	}
	ReportInfo.Dinfo	=	ReportDeviceInfo

	fmt.Println(ReportInfo)
	fmt.Println(" ")

	SendInfoProtobufStrThroughHttp(ReportInfo)
}

func MakeHypervisorInfoProtobufStructure (){

	var ReportInfo		=	&zmet.ZInfoMsg{}
	var cpu_count		=	2
	var memory_size		=	200
	var storage_size	=	1000

	hypervisorType := new(zmet.ZInfoTypes)
	*hypervisorType		=	zmet.ZInfoTypes_ZiHypervisor
	ReportInfo.Ztype	=	*hypervisorType

	ReportInfo.DevId	=	*proto.String("8f2238e7-948d-4601-a384-644c1b39467a")

	ReportHypervisorInfo := new(zmet.ZInfoHypervisor)
	ReportHypervisorInfo.Ncpu		=	*proto.Uint32(uint32(cpu_count))
	ReportHypervisorInfo.Memory		=	*proto.Uint64(uint64(memory_size))
	ReportHypervisorInfo.Storage	=	*proto.Uint64(uint64(storage_size))

	ReportDeviceSoftwareInfo := new(zmet.ZInfoSW)
	ReportDeviceSoftwareInfo.SwVersion	=	*proto.String("0.0.0.1")
	ReportDeviceSoftwareInfo.SwHash		=	*proto.String("jdjduu123")
	ReportHypervisorInfo.SwVersion		=	ReportDeviceSoftwareInfo

	ReportInfo.Hinfo	=	ReportHypervisorInfo

	fmt.Println(ReportInfo)
	fmt.Println(" ")

	SendInfoProtobufStrThroughHttp(ReportInfo)
}

func MakeAppInfoProtobufStructure (){

	var ReportInfo		=	&zmet.ZInfoMsg{}
	var cpu_count		=	2
	var memory_size		=	200
	var storage_size	=	1000

	appType := new(zmet.ZInfoTypes)
	*appType			=	zmet.ZInfoTypes_ZiApp
	ReportInfo.Ztype	=	*appType
	ReportInfo.DevId	=	*proto.String("8f2238e7-948d-4601-a384-644c1b39467a")

	ReportAppInfo := new(zmet.ZInfoApp)
	ReportAppInfo.AppID		=	*proto.String("8f2238e7-948d-4601-a384-644c1b39467")
	ReportAppInfo.Ncpu		=	*proto.Uint32(uint32(cpu_count))
	ReportAppInfo.Memory	=	*proto.Uint32(uint32(memory_size))
	ReportAppInfo.Storage	=	*proto.Uint32(uint32(storage_size))

	ReportVerInfo := new(zmet.ZInfoSW)
	ReportVerInfo.SwVersion		=	*proto.String("0.0.0.1")
	ReportVerInfo.SwHash		=	*proto.String("0.0.0.1")

	ReportAppInfo.SwVersion		=	ReportVerInfo
	ReportInfo.Ainfo			=	ReportAppInfo

	fmt.Println(ReportInfo)
	fmt.Println(" ")

	SendInfoProtobufStrThroughHttp(ReportInfo)
}

func SendInfoProtobufStrThroughHttp (ReportInfo *zmet.ZInfoMsg) {

	var ReportInfoAndMetricsToZedCloud = &zmsg.ZMsg{}
	var msgid = 1233

	ReportInfoAndMetricsToZedCloud.Msgid = *proto.Uint64(uint64(msgid))

	infoType := new(zmsg.ZMsgType)
	*infoType = zmsg.ZMsgType_ZInfo

	ReportInfoAndMetricsToZedCloud.Ztype	=	*infoType
	ReportInfoAndMetricsToZedCloud.Info		=	*ReportInfo
	fmt.Println(ReportInfoAndMetricsToZedCloud)

	data, err := proto.Marshal(ReportInfoAndMetricsToZedCloud)
	if err != nil {
		fmt.Println("marshaling error: ", err)
	}
	resp, err := http.Post(statusURL, "application/x-proto-binary",
		bytes.NewBuffer(data))
	if err != nil {
		fmt.Println(err)
	}
	res, err := ioutil.ReadAll(resp .Body)
	fmt.Println("response: ",res)

	/*newTest := &zmet.ZMsg{}
	err = proto.Unmarshal(data, newTest)
	if err != nil {
		log.Fatal("unmarshaling error: ", err)
	}

	log.Println(newTest)*/


}

func SendMetricsProtobufStrThroughHttp (ReportMetricsToZedCloud *zmet.ZMetricMsg) {

	var ReportInfoAndMetricsToZedCloud = &zmsg.ZMsg{}
	var msgid = 1234
	ReportInfoAndMetricsToZedCloud.Msgid = *proto.Uint64(uint64(msgid))

	metricType := new(zmsg.ZMsgType)
	*metricType = zmsg.ZMsgType_ZMetric
	ReportInfoAndMetricsToZedCloud.Ztype	=	*metricType
	ReportInfoAndMetricsToZedCloud.Metric	=	ReportMetricsToZedCloud
	fmt.Println(ReportInfoAndMetricsToZedCloud)

	data, err := proto.Marshal(ReportInfoAndMetricsToZedCloud)
	if err != nil {
		fmt.Println("marshaling error: ", err)
	}

	resp1, err := http.Post(metricsURL, "application/x-proto-binary",
		bytes.NewBuffer(data))
	if err != nil {
		fmt.Println(err)
	}
	res1, err := ioutil.ReadAll(resp1 .Body)
	fmt.Println("response metric: ",res1)

	/*newTest := &zmet.ZMsg{}
	err = proto.Unmarshal(data, newTest)
	if err != nil {
		log.Fatal("unmarshaling error: ", err)
	}

	log.Println(newTest)*/
}
