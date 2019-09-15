package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/astaxie/beego/logs"
	_ "github.com/lib/pq"
)

func main() {
	defaultDir, err := os.Getwd()
	if err != nil {
		logs.Error(err)
		return
	}
	var localpath = flag.String("p", defaultDir, "图片存储地址")

	now := time.Now()
	ftime := now.AddDate(0, 0, -1)
	defaultDate := ftime.Format("20060102")
	var date = flag.String("d", defaultDate, "日期，格式例如20060101，默认为前一天")
	flag.Parse()
	if exist := PathExists(*localpath); !exist {
		logs.Error("图片存储地址不存在")
		return
	}
	proxy := func(_ *http.Request) (*url.URL, error) {
		return url.Parse("socks5://127.0.0.1:1080")
	}

	transport := &http.Transport{Proxy: proxy}

	client := &http.Client{Transport: transport}

	geturl := "https://www.pixiv.net/ranking.php?mode=daily&content=illust&date=" + *date
	request, err := http.NewRequest("GET", geturl, nil)
	if err != nil {
		logs.Error("url is err ", err)
		return
	}
	request.Header.Add("accept-language", "zh-CN,zh;q=0.9")
	resp, err := client.Do(request)
	if err != nil {
		logs.Error("无法获取这一天的数据", err)
		return
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
		return
	}

	wrong := doc.Find(".error-unit")
	if exist := wrong.Text(); exist != "" {
		var errInfo string
		wrong.Find("p").EachWithBreak(func(i int, s *goquery.Selection) bool {
			errInfo = s.Text()
			return false

		})
		logs.Info(errInfo)
		return
	}

	list := doc.Find(".ranking-items")

	//图片id列表
	var ids []string
	list.Find("section").Each(func(i int, s *goquery.Selection) {
		//图片id
		id, _ := s.Attr("data-id")
		ids = append(ids, id)

	})

	err = os.Mkdir(*localpath+"/"+*date, os.ModePerm)
	if err != nil {
		logs.Error(err)
		return
	}

	for k, v := range ids {

		detailURL := "https://www.pixiv.net/member_illust.php?mode=medium&illust_id=" + v
		resp, err := client.Get(detailURL)
		if err != nil {
			logs.Error(err)
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logs.Error(err)
			return
		}
		index := strings.Index(string(body), "urls")
		var tmp []byte
		for i := index; i < len(body); i++ {
			tmp = append(tmp, body[i])
		}

		index = strings.Index(string(tmp), "}")
		var tmp1 []byte
		for i := 0; i <= index; i++ {
			tmp1 = append(tmp1, tmp[i])
		}

		index = strings.Index(string(tmp1), "{")
		var ansStr []byte
		for i := index; i < len(tmp1); i++ {
			ansStr = append(ansStr, tmp1[i])
		}

		var dat map[string]interface{}
		err = json.Unmarshal(ansStr, &dat)
		if err != nil {
			logs.Error(err)
			return
		}
		//获取最终url
		finURL := dat["original"].(string)
		logs.Info(finURL)

		//发送请求得到图片

		//设置请求头
		request, err := http.NewRequest("GET", finURL, nil)
		if err != nil {
			logs.Error("url is err ", err)
			return
		}
		request.Header.Add("Referer", detailURL)

		//发送请求获取结果
		resp, err = client.Do(request)
		if err != nil {
			logs.Error(err)
			return
		}
		if resp != nil {
			defer resp.Body.Close()
		}
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			logs.Error(err)
			return
		}

		var fileName string
		if strings.HasSuffix(finURL, "png") {
			fileName = *localpath + "/" + *date + "/第" + strconv.Itoa(k+1) + "名.png"
		} else {
			fileName = *localpath + "/" + *date + "/第" + strconv.Itoa(k+1) + "名.jpg"
		}
		logs.Info(fileName)

		f, err := os.Create(fileName)
		defer f.Close()
		if err != nil {
			logs.Error(err)
			return
		}
		_, err = f.Write(body)
		if err != nil {
			logs.Error(err)
			return
		}

	}
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}
