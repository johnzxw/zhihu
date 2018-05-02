package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	CurrentPath        string
	requestLogFileName string
	logFileName        string
	listerAddr         string
	gzipLevel          int
	readTimeout        time.Duration
	writeTimeout       time.Duration
	hideBanner         bool
	hidePort           bool
	debug              bool
	apiUrl             string
	apiTodayUrl        string
	dbFileName         string
	tableName          string
	SqlConnect         *sql.DB
)

type ApiDataStruct struct {
	Date        string              `json:"date"`
	News        []ContentDataStruct `json:"news"`
	Displaydate string              `json:"display_date"`
}

type ContentDataStruct struct {
	Title     string `json:"title"`
	Url       string `json:"url"`
	Image     string `json:"image"`
	Shareurl  string `json:"share_url"`
	Thumbnail string `json:"thumbnail"`
	Gaprefix  string `json:"ga_prefix"`
	Id        int    `json:"id"`
}
type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

/**
 *初始化变量
 */
func init() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic("currentpath error")
	}
	CurrentPath = strings.Replace(dir, "\\", "/", -1)

	requestLogFileName = "/request.log"
	logFileName = "/go_debug_log.log"
	listerAddr = "127.0.0.1:1101"
	gzipLevel = 5
	readTimeout = 10
	writeTimeout = 10
	hideBanner = true
	hidePort = true
	debug = true
	apiUrl = "http://news.at.zhihu.com/api/1.2/news/before/"
	apiTodayUrl = "http://news.at.zhihu.com/api/1.2/news/latest"
	dbFileName = CurrentPath + "/zhihudb.db"
	tableName = "zhihudata"

	db, err := sql.Open("sqlite3", dbFileName)
	if err != nil {
		panic(err)
	}
	SqlConnect = db

	//executeSql()
}
func main() {
	t := &Template{
		templates: template.Must(template.ParseGlob(CurrentPath + "/*.html")),
	}
	e := echo.New()
	e.HideBanner = hideBanner
	e.HidePort = hidePort
	e.Debug = debug
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: gzipLevel,
	}))

	//把请求日志放到文件中
	requestLogFile, err := os.OpenFile(CurrentPath+requestLogFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0755)
	defer requestLogFile.Close()
	if err != nil {
		e.Logger.Fatal("打开请求日志输出文件失败！ 请检查文件权限。" + requestLogFileName + "  error !!")
	}
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time_unix":"${time_unix}","time_unix_nano":"${time_unix_nano}","time_rfc3339":"${time_rfc3339}","time_rfc3339_nano":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}","url":"${uri}","host":"${host}","method":"${method}","path":"${path}","referer":"${referer}","user_agent":"${user_agent}","status":"${status}","latency":"${latency}","latency_human":"${latency_human}","bytes_in":"${bytes_in}","bytes_out":"${bytes_out}","header":"${header}","query":"${query}","form":"${form}"}` + "\n",
		Output: requestLogFile,
	}))

	//把错误日志放到文件中
	logFile, err := os.OpenFile(CurrentPath+logFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0755)
	defer logFile.Close()
	if err != nil {
		e.Logger.Fatal("打开日志输出文件失败！ 请检查文件权限。" + logFileName + "  error !!")
	}
	//日志输出到文件
	e.Logger.SetLevel(log.INFO)
	e.Logger.SetOutput(logFile)
	e.Renderer = t
	e.HTTPErrorHandler = EchoErrorHandle
	e.DisableHTTP2 = true
	e.Server.ReadTimeout = readTimeout * time.Second
	e.Server.WriteTimeout = writeTimeout * time.Second
	//静态文件
	e.File("/favicon.ico", CurrentPath+"/static/favicon.ico")
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root: CurrentPath,
		//Browse: true,
		//HTML5:  true,
	}))

	//route
	e.GET("/", DefaultIndex)
	e.POST("/getdata", DefaultIndex)

	e.Server.Addr = listerAddr
	e.Logger.Fatal(e.Start(listerAddr))
}

/**
 *首页
 */
func DefaultIndex(c echo.Context) error {
	date := c.FormValue("date")
	if isPost(c.Request().Method) && len(date) > 0 {
		return c.JSON(http.StatusOK, getDataByDate(date))
	}
	return c.Render(http.StatusOK, "index.html", map[string]string{})
}

/**
 *根据日期获取数据
 */
func getDataByDate(date string) []ContentDataStruct {
	var zhihuData []ContentDataStruct
	_, err := time.Parse("20060102", date)
	if err == nil {
		zhihuData = getDataForDb(date)
		//走接口获取数据
		if len(zhihuData) < 1 {
			currentYMD := time.Unix(time.Now().Unix(), 0).Format("20060102")
			var url string
			if date == currentYMD {
				url = apiTodayUrl
			} else {
				url = apiUrl + date
			}
			data := httpGet(url)
			resultArray := &ApiDataStruct{}
			errs := json.Unmarshal([]byte(data), resultArray)
			if errs != nil {
				fmt.Println("json 解析失败！ url：")
			}
			zhihuData = resultArray.News
			//对url进行处理
			for i := 0; i < len(zhihuData); i++ {
				zhihuData[i].Thumbnail = strings.Replace(zhihuData[i].Thumbnail, "http:", "", 1)
				zhihuData[i].Shareurl = strings.Replace(zhihuData[i].Shareurl, "http:", "", 1)
				zhihuData[i].Image = strings.Replace(zhihuData[i].Image, "http:", "", 1)
				zhihuData[i].Url = strings.Replace(zhihuData[i].Url, "http:", "", 1)
			}
			saveDataToDb(zhihuData, date)
		}
	}
	return zhihuData
}

/**
 * 保存数据到sqlite
 */
func saveDataToDb(data []ContentDataStruct, date string) bool {
	//defer SqlConnect.Close()
	for _, data := range data {
		stmt, err := SqlConnect.Prepare("INSERT INTO " + tableName + " (Title,Url,Image,Shareurl,Thumbnail,Gaprefix,Id,Date) values (?,?,?,?,?,?,?,?)")
		if err != nil {
			panic(err)
		}

		stmt.Exec(data.Title, data.Url, data.Image, data.Shareurl, data.Thumbnail, data.Gaprefix, data.Id, date)
	}
	return true
}

/**
 *走库获取数据
 */
func getDataForDb(date string) []ContentDataStruct {
	var returnData []ContentDataStruct
	rows, err := SqlConnect.Query("SELECT Title,Url,Image,Shareurl,Thumbnail,Gaprefix,Id FROM "+tableName+" where Date = ?",
		date)
	if err != nil {
		panic(err)
	}
	//defer SqlConnect.Close()
	for rows.Next() {
		var Title string
		var Url string
		var Image string
		var Shareurl string
		var Thumbnail string
		var Gaprefix string
		var Id int
		err = rows.Scan(&Title, &Url, &Image, &Shareurl, &Thumbnail, &Gaprefix, &Id)
		if err != nil {
			panic(err)
		}
		returnData = append(returnData, ContentDataStruct{Title, Url, Image, Shareurl, Thumbnail, Gaprefix, Id})
	}
	return returnData
}

/**
 *http get
 */
func httpGet(url string) string {
	client := &http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		return ""
	}
	//Add 头协议
	request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Add("Accept-Language", "ja,zh-CN;q=0.8,zh;q=0.6")
	request.Header.Add("Connection", "keep-alive")
	request.Header.Add("Cookie", "")
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:12.0) Gecko/20100101 Firefox/12.0")
	//接收服务端返回给客户端的信息
	response, _ := client.Do(request)
	defer response.Body.Close()

	if response.StatusCode == 200 {
		str, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Fatal error ", err.Error())
			return ""
		}
		return string(str)
	} else {
		return ""
	}
}

/**
 *判断提交方法是否是post
 */
func isPost(method string) bool {
	result := false
	if method == http.MethodPost {
		result = true
	}
	return result
}

/**
 *错误处理
 */
func EchoErrorHandle(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	errorPage := fmt.Sprintf("%d.html", code)
	c.Logger().Error("errorpage:" + errorPage)
	c.Logger().Error(err)
}
