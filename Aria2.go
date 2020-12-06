package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"v2/rpc"
)

var aria2Rpc rpc.Client
var aria2Set Aria2DownloadMode

// TMMessageChan is set Torrent/Magnet download mode
var TMMessageChan = make(chan string, 3)

// TMAllowDownloads is Allow Torrent/Magnet download List
var TMAllowDownloads = make(map[string]int, 0)

func testTMStop() {
	for {
		gid := <-TMMessageChan
		//log.Println(a)
		if _, have := TMAllowDownloads[gid]; !have {
			dlInfo, err := aria2Rpc.TellStatus(gid)
			dropErr(err)
			//log.Printf("%+v\n", dlInfo)
			if dlInfo.BitTorrent.Info.Name != "" && aria2Set.TMMode != "3" {
				aria2Rpc.Pause(gid)
				TMSelectMessageChan <- gid
			}
		} else {
			delete(TMAllowDownloads, gid)
		}

	}

}
func aria2Load() {
	var err error
	aria2Rpc, err = rpc.New(context.Background(), info.Aria2Server, info.Aria2Key, time.Second*10, &Aria2Notifier{})
	dropErr(err)
	aria2Set = Aria2DownloadMode{
		TMMode: "1", // 记得修改
	}
	log.Printf(locText("connectTo"), info.Aria2Server)
	version, err := aria2Rpc.GetVersion()
	dropErr(err)
	log.Printf(locText("connectSuccess"), version.Version)
	go testTMStop()
}

func formatTellSomething(info []rpc.StatusInfo, err error) string {
	dropErr(err)
	res := ""
	var statusFlag = map[string]string{"active": locText("active"), "paused": locText("paused"), "complete": locText("complete"), "removed": locText("removed")}
	for index, Files := range info {
		if Files.BitTorrent.Info.Name != "" {
			m := make(map[string]string)
			//paths, fileName := filepath.Split(files)
			m["GID"] = Files.Gid
			m["Name"] = Files.BitTorrent.Info.Name
			bytes, err := strconv.ParseFloat(Files.TotalLength, 64)
			dropErr(err)
			m["Size"] = byte2Readable(bytes)
			completedLength, err := strconv.ParseFloat(Files.CompletedLength, 64)
			dropErr(err)
			m["Progress"] = strconv.FormatFloat(completedLength*100.0/bytes, 'f', 2, 64) + " %"
			// m["Threads"] = fmt.Sprint(len(File.URIs))
			downloadSpeed, err := strconv.ParseFloat(Files.DownloadSpeed, 64)
			dropErr(err)
			m["Speed"] = byte2Readable(downloadSpeed)
			m["Status"] = statusFlag[Files.Status]
			if Files.Status == "paused" {
				res += fmt.Sprintf(locText("queryInformationFormat1"), m["GID"], m["Name"], m["Progress"], m["Size"])
			} else if Files.Status == "complete" || Files.Status == "removed" {
				res += fmt.Sprintf(locText("queryInformationFormat2"), m["GID"], m["Name"], m["Status"], m["Progress"], m["Size"])
			} else {
				res += fmt.Sprintf(locText("queryInformationFormat3"), m["GID"], m["Name"], m["Progress"], m["Size"], m["Speed"])
			}
		} else {
			for _, File := range Files.Files {
				m := make(map[string]string)
				//paths, fileName := filepath.Split(files)
				m["GID"] = Files.Gid
				countSplit := strings.Split(File.Path, "/")
				m["Name"] = countSplit[len(countSplit)-1]
				bytes, err := strconv.ParseFloat(Files.TotalLength, 64)
				dropErr(err)
				m["Size"] = byte2Readable(bytes)
				completedLength, err := strconv.ParseFloat(Files.CompletedLength, 64)
				dropErr(err)
				m["Progress"] = strconv.FormatFloat(completedLength*100.0/bytes, 'f', 2, 64) + " %"
				m["Threads"] = fmt.Sprint(len(File.URIs))
				downloadSpeed, err := strconv.ParseFloat(Files.DownloadSpeed, 64)
				dropErr(err)
				m["Speed"] = byte2Readable(downloadSpeed)
				m["Status"] = statusFlag[Files.Status]
				if Files.Status == "paused" {
					res += fmt.Sprintf(locText("queryInformationFormat1"), m["GID"], m["Name"], m["Progress"], m["Size"])
				} else if Files.Status == "complete" || Files.Status == "removed" {
					res += fmt.Sprintf(locText("queryInformationFormat2"), m["GID"], m["Name"], m["Status"], m["Progress"], m["Size"])
				} else {
					res += fmt.Sprintf(locText("queryInformationFormat3"), m["GID"], m["Name"], m["Progress"], m["Size"], m["Speed"])
				}
			}
		}

		if index != len(info) {
			res += "\n\n"
		}
	}
	return res
}

func formatGidAndName(info []rpc.StatusInfo, err error) []map[string]string {
	dropErr(err)

	m := make([]map[string]string, 0)
	//log.Printf("%+v\n", info)
	for _, Files := range info {
		// log.Println(Files.BitTorrent.Info.Name, Files.BitTorrent.Info.Name != "")
		if Files.BitTorrent.Info.Name != "" {
			ms := make(map[string]string)
			ms["GID"] = Files.Gid
			ms["Name"] = Files.BitTorrent.Info.Name
			m = append(m, ms)
		} else {
			for _, File := range Files.Files {
				ms := make(map[string]string)
				//paths, fileName := filepath.Split(files)
				ms["GID"] = Files.Gid
				countSplit := strings.Split(File.Path, "/")
				ms["Name"] = countSplit[len(countSplit)-1]
				m = append(m, ms)
			}
		}
	}
	return m
}

func tellName(info rpc.StatusInfo, err error) string {
	dropErr(err)
	//log.Printf("%+v\n", info)
	Name := ""
	if info.BitTorrent.Info.Name != "" {
		Name = info.BitTorrent.Info.Name
	} else {
		for _, File := range info.Files {
			if File.Path != "" {
				countSplit := strings.Split(File.Path, "/")
				Name = countSplit[len(countSplit)-1]
			} else {
				Name = info.Gid
			}
		}
	}
	return Name
}

func download(uri string) bool {
	uriType := isDownloadType(uri)
	if uriType == 0 {
		return false
	}
	uriData := make([]string, 0)
	uriData = append(uriData, uri)
	switch uriType {
	case 1:
		aria2Rpc.AddURI(uriData)
	case 2:
		aria2Rpc.AddURI(uriData)
	case 3:
		aria2Rpc.AddTorrent(uri)
	}
	return true
}

func formatTMFiles(gid string) [][]string {
	var fileList [][]string
	rawList, err := aria2Rpc.GetFiles(gid)
	dropErr(err)
	// log.Printf("%+v", rawList)
	for _, file := range rawList {
		fileInfo := make([]string, 0)
		fileInfo = append(fileInfo, file.Path)
		bytes, err := strconv.ParseFloat(file.Length, 64)
		dropErr(err)
		fileInfo = append(fileInfo, byte2Readable(bytes))
		fileInfo = append(fileInfo, file.Selected)
		fileList = append(fileList, fileInfo)
	}
	return fileList
}

func setTMDownloadFilesAndStart(gid string, FilesList [][]string) {
	selectFile := ""
	for i, file := range FilesList {
		if file[2] == "true" {
			selectFile += fmt.Sprint(i+1) + ","
		}
	}
	aria2Rpc.ChangeOption(gid, rpc.Option{
		"select-file": selectFile[:len(selectFile)-1],
	})
	TMAllowDownloads[gid] = 0
	aria2Rpc.Unpause(gid)

}
func selectBigestFile(gid string) int {
	index := 0
	rawList, err := aria2Rpc.GetFiles(gid)
	dropErr(err)
	for i := 0; i < len(rawList); i++ {
		if rawList[i].Length > rawList[index].Length {
			index = i
		}
	}
	return index
}
func selectBigFiles(gid string) []int {
	indexs := make([]int, 0)
	rawList, err := aria2Rpc.GetFiles(gid)
	dropErr(err)
	totalSize, avgSize := 0, 0.0
	for _, file := range rawList {
		totalSize += toInt(file.Length)
	}
	avgSize = float64(totalSize) / float64(len(rawList))
	avgSize -= avgSize * 0.2

	for i := 0; i < len(rawList); i++ {
		dist, err := strconv.ParseFloat(rawList[i].Length, 64)
		dropErr(err)
		if dist > avgSize {
			indexs = append(indexs, i)
		}
	}
	return indexs
}
