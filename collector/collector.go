/**
 * @Author xiaoxiao
 * @Description CREATE FILE collector
 * @Date 2020/10/10 10:29 上午
 **/
package collector

import (
	"FetchPPT/util"
	"bytes"
	"fmt"
	"github.com/gocolly/colly/v2"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

const (
	originUrl         = "http://www.di1ppt.com"
	downloadOriginUrl = "http://www.di1ppt.com/e/DownSys"
)

type SeedConfig struct {
	DownloadFile    DownloadFile
	GetNavCollector func(file DownloadFile)
}

type DownloadFile struct {
	Filename        string
	Url             string
	Directory       string
	OriginDirectory string
}

type CrawCollector struct {
	DownloadFile DownloadFile
	Wg           *sync.WaitGroup
}

var filterMap = map[string]bool{
	"/office/":             true,
	"/ziti/":               true,
	"http://www.10103.com": true,
}

var page_list = make(map[string]bool)

func Run() {
	SeedConfig := SeedConfig{
		DownloadFile: DownloadFile{
			Filename:  "",
			Url:       "",
			Directory: "",
		},
		GetNavCollector: GetNavCollector,
	}
	SeedConfig.Run()
}

func (s *SeedConfig) Run() {
	s.GetNavCollector(s.DownloadFile)
}

var Wg = &sync.WaitGroup{}

//获取导航页面
func GetNavCollector(downloadFile DownloadFile) {
	c := colly.NewCollector()

	c.OnHTML("#navMenu  li", func(element *colly.HTMLElement) {
		cc := &CrawCollector{
			DownloadFile: downloadFile,
		}
		seedUrl := element.ChildAttr("a", "href")
		if !filterMap[seedUrl] {

			directory := element.ChildText("a>span")
			dir := path.Join(downloadFile.OriginDirectory, directory)
			if !util.Exists(dir) {
				os.MkdirAll(dir, 0777)
			}

			cc.DownloadFile.Directory = dir
			Wg.Add(1)
			cc.GetDetailCollector(seedUrl)

		}

	})
	c.OnScraped(func(response *colly.Response) {

		fmt.Println("完成全部抓取")
	})
	c.Visit(originUrl)
	Wg.Wait()
}

//获取导航对应首页N页列表
func (cc *CrawCollector) GetDetailCollector(seedUrl string) {

	c := colly.NewCollector()
	c.OnHTML(".dlbox .clearfix .pages", func(element *colly.HTMLElement) {
		lis := element.DOM.Find("li")
		pageUrl, _ := lis.Last().Find("a").Attr("href")
		split := strings.Split(pageUrl, "_")
		ii, _ := strconv.Atoi(strings.Split(split[1], ".")[0])

		for i := 1; i < ii+1; i++ {
			url := seedUrl + "index_" + strconv.Itoa(i) + ".html"
			cc.GetPageDetailCollector(url)
		}

	})

	c.Visit(originUrl + seedUrl)
}

//获取每页对应的详情页
func (cc *CrawCollector) GetPageDetailCollector(seedUrl2 string) {
	c := colly.NewCollector()

	c.OnHTML(".dlbox .tplist li>a", func(element *colly.HTMLElement) {

		detailUrl := element.Attr("href")
		Wg.Add(1)
		go cc.GetDownloadUrlCollector(detailUrl)

	})
	c.Visit(originUrl + seedUrl2)
}

//获取下载页面
func (cc *CrawCollector) GetDownloadUrlCollector(detailUrl string) {
	c := colly.NewCollector()

	c.OnHTML(".downurllist li>a", func(element *colly.HTMLElement) {
		if element.Index == 0 {
			downloadlUrl := element.Attr("href")

			cc.GetDownloadUrlDetailCollector(downloadlUrl)
		}

	})
	c.Visit(originUrl + detailUrl)
}

//获取验证码下载页面
func (cc *CrawCollector) GetDownloadUrlDetailCollector(downLoadDetailUrl string) {
	c := colly.NewCollector()
	c.OnHTML("tbody td>a", func(element *colly.HTMLElement) {

		downloadUrl := element.Attr("href")
		downloadUrl = strings.Replace(downloadUrl, "..", "", -1)
		cc.DownloadFile.Url = downloadOriginUrl + downloadUrl
		Wg.Add(1)
		go cc.FetchPPT(downloadOriginUrl + downloadUrl)
	})
	c.Visit(originUrl + downLoadDetailUrl)
}

//获取ppt详情下载页面
func (cc *CrawCollector) FetchPPT(dowloadUrl string) {
	c := colly.NewCollector()
	defer Wg.Done()
	c.OnResponse(func(response *colly.Response) {
		filename := response.FileName()

		filepath := path.Join(cc.DownloadFile.Directory, filename)

		if _, err := os.Stat(filepath); err == nil {
			log.Println("文件已存在:", filename)
			return
		}
		output, err := os.Create(filepath)
		defer output.Close()
		if err != nil {
			log.Println("创建失败: ", err)
		}
		_, err = io.Copy(output, bytes.NewReader(response.Body))
		if err != nil {
			log.Println("写入失败 ", err)
		}
		log.Printf("下载文件 %s/%s", cc.DownloadFile.Directory, filename)
	})
	c.Visit(dowloadUrl)
}
