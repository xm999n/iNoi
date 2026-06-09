package _123

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// do others that not defined in Driver interface

const (
	ApiBase           = "https://www.123pan.com"
	Api               = ApiBase + "/api"
	AApi              = ApiBase + "/a/api"
	BApi              = ApiBase + "/b/api"
	LoginApi          = "https://login.123pan.com/api"
	MainApi           = BApi
	SignInAndroid     = BApi + "/user/sign_in"
	SignInWeb         = LoginApi + "/user/sign_in"
	Logout            = MainApi + "/user/logout"
	UserInfo          = BApi + "/user/info"
	FileList          = Api + "/file/list/new"
	DownloadInfo      = AApi + "/file/download_info"
	Mkdir             = AApi + "/file/upload_request"
	Move              = MainApi + "/file/mod_pid"
	Rename            = MainApi + "/file/rename"
	Trash             = AApi + "/file/trash"
	UploadRequest     = BApi + "/file/upload_request"
	UploadComplete    = BApi + "/file/upload_complete"
	S3PreSignedUrls   = BApi + "/file/s3_repare_upload_parts_batch"
	S3Auth            = BApi + "/file/s3_upload_object/auth"
	UploadCompleteV2  = BApi + "/file/upload_complete/v2"
	S3Complete        = BApi + "/file/s3_complete_multipart_upload"
	OfflineResolve    = MainApi + "/v2/offline_download/task/resolve"
	OfflineSubmit     = MainApi + "/v2/offline_download/task/submit"
	OfflineTaskList   = MainApi + "/offline_download/task/list"
	OfflineTaskDelete = MainApi + "/offline_download/task/delete"
	// AuthKeySalt      = "8-8D$sL8gPjom7bk#cY"
)

var ErrOfflineTaskNotFound = errors.New("offline task not found")

const (
	androidAppVersion   = "61"
	androidXAppVersion  = "2.4.0"
	androidDeviceBrand  = "Xiaomi"
	androidPlatformName = "android"
)

var (
	androidDeviceTypes = []string{
		"24075RP89G", "24076RP19G", "24076RP19I", "M1805E10A", "M2004J11G",
		"M2012K11AG", "M2104K10I", "22021211RG", "22021211RI", "21121210G",
		"23049PCD8G", "23049PCD8I", "23013PC75G", "24069PC21G", "24069PC21I",
		"23113RKC6G", "M1912G7BI", "M2007J20CI", "M2007J20CG", "M2007J20CT",
		"M2102J20SG", "M2102J20SI", "21061110AG", "2201116PG", "2201116PI",
		"22041216G", "22041216UG", "22111317PG", "22111317PI", "22101320G",
		"22101320I", "23122PCD1G", "23122PCD1I", "2311DRK48G", "2311DRK48I",
		"2312FRAFDI", "M2004J19PI",
	}
	androidOSVersions = []string{
		"Android_7.1.2", "Android_8.0.0", "Android_8.1.0", "Android_9.0",
		"Android_10", "Android_11", "Android_12", "Android_13",
		"Android_6.0.1", "Android_5.1.1", "Android_4.4.4", "Android_4.3",
		"Android_4.2.2", "Android_4.1.2",
	}
	androidRand       = rand.New(rand.NewSource(time.Now().UnixNano()))
	androidDeviceType = pickAndroid(androidDeviceTypes)
	androidOSVersion  = pickAndroid(androidOSVersions)
	androidLoginUUID  = randHex(32)
)

func pickAndroid(options []string) string {
	if len(options) == 0 {
		return ""
	}
	return options[androidRand.Intn(len(options))]
}

func randHex(n int) string {
	const hex = "0123456789abcdef"
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[androidRand.Intn(len(hex))]
	}
	return string(b)
}

func useAndroidProtocol() bool {
	return strings.ToLower(strings.TrimSpace(os.Getenv("PAN123_PROTOCOL"))) != "web"
}

func androidHeaders(authorization string) map[string]string {
	return map[string]string{
		"content-type":    "application/json",
		"authorization":   authorization,
		"LoginUuid":       androidLoginUUID,
		"user-agent":      fmt.Sprintf("123pan/v%s(%s;%s)", androidXAppVersion, androidOSVersion, androidDeviceBrand),
		"accept-encoding": "gzip",
		"osversion":       androidOSVersion,
		"platform":        androidPlatformName,
		"devicetype":      androidDeviceType,
		"devicename":      androidDeviceBrand,
		"host":            "www.123pan.com",
		"app-version":     androidAppVersion,
		"x-app-version":   androidXAppVersion,
	}
}

func webHeaders(authorization string) map[string]string {
	return map[string]string{
		"origin":        "https://www.123pan.com",
		"referer":       "https://www.123pan.com/",
		"authorization": authorization,
		"user-agent":    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) openlist-client",
		"platform":      "web",
		"app-version":   "3",
	}
}

func signPath(path string, os string, version string) (k string, v string) {
	table := []byte{'a', 'd', 'e', 'f', 'g', 'h', 'l', 'm', 'y', 'i', 'j', 'n', 'o', 'p', 'k', 'q', 'r', 's', 't', 'u', 'b', 'c', 'v', 'w', 's', 'z'}
	random := fmt.Sprintf("%.f", math.Round(1e7*rand.Float64()))
	now := time.Now().In(time.FixedZone("CST", 8*3600))
	timestamp := fmt.Sprint(now.Unix())
	nowStr := []byte(now.Format("200601021504"))
	for i := 0; i < len(nowStr); i++ {
		nowStr[i] = table[nowStr[i]-48]
	}
	timeSign := fmt.Sprint(crc32.ChecksumIEEE(nowStr))
	data := strings.Join([]string{timestamp, random, path, os, version, timeSign}, "|")
	dataSign := fmt.Sprint(crc32.ChecksumIEEE([]byte(data)))
	return timeSign, strings.Join([]string{timestamp, random, dataSign}, "-")
}

func GetApi(rawUrl string) string {
	u, _ := url.Parse(rawUrl)
	query := u.Query()
	query.Add(signPath(u.Path, "web", "3"))
	u.RawQuery = query.Encode()
	return u.String()
}

//func GetApi(url string) string {
//	vm := js.New()
//	vm.Set("url", url[22:])
//	r, err := vm.RunString(`
//	(function(e){
//        function A(t, e) {
//            e = 1 < arguments.length && void 0 !== e ? e : 10;
//            for (var n = function() {
//                for (var t = [], e = 0; e < 256; e++) {
//                    for (var n = e, r = 0; r < 8; r++)
//                        n = 1 & n ? 3988292384 ^ n >>> 1 : n >>> 1;
//                    t[e] = n
//                }
//                return t
//            }(), r = function(t) {
//                t = t.replace(/\\r\\n/g, "\\n");
//                for (var e = "", n = 0; n < t.length; n++) {
//                    var r = t.charCodeAt(n);
//                    r < 128 ? e += String.fromCharCode(r) : e = 127 < r && r < 2048 ? (e += String.fromCharCode(r >> 6 | 192)) + String.fromCharCode(63 & r | 128) : (e = (e += String.fromCharCode(r >> 12 | 224)) + String.fromCharCode(r >> 6 & 63 | 128)) + String.fromCharCode(63 & r | 128)
//                }
//                return e
//            }(t), a = -1, i = 0; i < r.length; i++)
//                a = a >>> 8 ^ n[255 & (a ^ r.charCodeAt(i))];
//            return (a = (-1 ^ a) >>> 0).toString(e)
//        }
//
//	   function v(t) {
//	       return (v = "function" == typeof Symbol && "symbol" == typeof Symbol.iterator ? function(t) {
//	                   return typeof t
//	               }
//	               : function(t) {
//	                   return t && "function" == typeof Symbol && t.constructor === Symbol && t !== Symbol.prototype ? "symbol" : typeof t
//	               }
//	       )(t)
//	   }
//
//		for (p in a = Math.round(1e7 * Math.random()),
//		o = Math.round(((new Date).getTime() + 60 * (new Date).getTimezoneOffset() * 1e3 + 288e5) / 1e3).toString(),
//		m = ["a", "d", "e", "f", "g", "h", "l", "m", "y", "i", "j", "n", "o", "p", "k", "q", "r", "s", "t", "u", "b", "c", "v", "w", "s", "z"],
//		u = function(t, e, n) {
//			var r;
//			n = 2 < arguments.length && void 0 !== n ? n : 8;
//			return 0 === arguments.length ? null : (r = "object" === v(t) ? t : (10 === "".concat(t).length && (t = 1e3 * Number.parseInt(t)),
//			new Date(t)),
//			t += 6e4 * new Date(t).getTimezoneOffset(),
//			{
//				y: (r = new Date(t + 36e5 * n)).getFullYear(),
//				m: r.getMonth() + 1 < 10 ? "0".concat(r.getMonth() + 1) : r.getMonth() + 1,
//				d: r.getDate() < 10 ? "0".concat(r.getDate()) : r.getDate(),
//				h: r.getHours() < 10 ? "0".concat(r.getHours()) : r.getHours(),
//				f: r.getMinutes() < 10 ? "0".concat(r.getMinutes()) : r.getMinutes()
//			})
//		}(o),
//		h = u.y,
//		g = u.m,
//		l = u.d,
//		c = u.h,
//		u = u.f,
//		d = [h, g, l, c, u].join(""),
//		f = [],
//		d)
//			f.push(m[Number(d[p])]);
//		return h = A(f.join("")),
//		g = A("".concat(o, "|").concat(a, "|").concat(e, "|").concat("web", "|").concat("3", "|").concat(h)),
//		"".concat(h, "=").concat(o, "-").concat(a, "-").concat(g);
//	})(url)
//	   `)
//	if err != nil {
//		fmt.Println(err)
//		return url
//	}
//	v, _ := r.Export().(string)
//	return url + "?" + v
//}

func (d *Pan123) login() error {
	var body base.Json
	if utils.IsEmailFormat(d.Username) {
		body = base.Json{
			"mail":     d.Username,
			"password": d.Password,
			"type":     2,
		}
	} else {
		if useAndroidProtocol() {
			body = base.Json{
				"passport": d.Username,
				"password": d.Password,
				"type":     1,
			}
		} else {
			body = base.Json{
				"passport": d.Username,
				"password": d.Password,
				"remember": true,
			}
		}
	}
	loginUrl := SignInWeb
	headers := webHeaders("")
	if useAndroidProtocol() {
		loginUrl = SignInAndroid
		headers = androidHeaders("")
	}
	res, err := base.RestyClient.R().
		SetHeaders(headers).
		SetBody(body).Post(loginUrl)
	if err != nil {
		return err
	}
	if utils.Json.Get(res.Body(), "code").ToInt() != 200 {
		err = fmt.Errorf(utils.Json.Get(res.Body(), "message").ToString())
	} else {
		d.AccessToken = utils.Json.Get(res.Body(), "data", "token").ToString()
	}
	return err
}

//func authKey(reqUrl string) (*string, error) {
//	reqURL, err := url.Parse(reqUrl)
//	if err != nil {
//		return nil, err
//	}
//
//	nowUnix := time.Now().Unix()
//	random := rand.Intn(0x989680)
//
//	p4 := fmt.Sprintf("%d|%d|%s|%s|%s|%s", nowUnix, random, reqURL.Path, "web", "3", AuthKeySalt)
//	authKey := fmt.Sprintf("%d-%d-%x", nowUnix, random, md5.Sum([]byte(p4)))
//	return &authKey, nil
//}

func (d *Pan123) Request(url string, method string, callback base.ReqCallback, resp interface{}) ([]byte, error) {
	isRetry := false
do:
	req := base.RestyClient.R()
	authorization := "Bearer " + d.AccessToken
	headers := webHeaders(authorization)
	if useAndroidProtocol() {
		headers = androidHeaders(authorization)
	}
	req.SetHeaders(headers)
	if callback != nil {
		callback(req)
	}
	if resp != nil {
		req.SetResult(resp)
	}
	//authKey, err := authKey(url)
	//if err != nil {
	//	return nil, err
	//}
	//req.SetQueryParam("auth-key", *authKey)
	finalURL := url
	if !useAndroidProtocol() {
		finalURL = GetApi(url)
	}
	res, err := req.Execute(method, finalURL)
	if err != nil {
		return nil, err
	}
	body := res.Body()
	code := utils.Json.Get(body, "code").ToInt()
	if code != 0 {
		if !isRetry && code == 401 {
			err := d.login()
			if err != nil {
				return nil, err
			}
			isRetry = true
			goto do
		}
		return nil, errors.New(jsoniter.Get(body, "message").ToString())
	}
	return body, nil
}

func (d *Pan123) OfflineDownload(ctx context.Context, uri string, dstDir model.Obj) (int64, error) {
	var resolveResp offlineResolveResp
	_, err := d.Request(OfflineResolve, http.MethodPost, func(req *resty.Request) {
		req.SetContext(ctx).SetBody(base.Json{
			"urls": uri,
		})
	}, &resolveResp)
	if err != nil {
		return 0, err
	}
	if len(resolveResp.Data.List) == 0 {
		return 0, fmt.Errorf("offline resolve failed: empty response")
	}
	if resolveResp.Data.List[0].Result != 0 {
		msg := resolveResp.Data.List[0].ErrMsg
		if msg == "" {
			msg = "offline resolve failed"
		}
		return 0, fmt.Errorf("%s", msg)
	}
	resourceID := resolveResp.Data.List[0].ID
	if resourceID == 0 {
		return 0, fmt.Errorf("offline resolve failed: empty resource id")
	}
	selectFileIDs := make([]int64, 0, len(resolveResp.Data.List[0].Files))
	for _, f := range resolveResp.Data.List[0].Files {
		if f.ID > 0 {
			selectFileIDs = append(selectFileIDs, f.ID)
		}
	}
	if len(selectFileIDs) == 0 {
		return 0, fmt.Errorf("offline resolve failed: empty file list")
	}
	uploadDir, err := strconv.ParseInt(dstDir.GetID(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid destination dir id: %s", dstDir.GetID())
	}

	var submitResp offlineSubmitResp
	_, err = d.Request(OfflineSubmit, http.MethodPost, func(req *resty.Request) {
		req.SetContext(ctx).SetBody(base.Json{
			"resource_list": []base.Json{
				{
					"resource_id":    resourceID,
					"select_file_id": selectFileIDs,
				},
			},
			"upload_dir": uploadDir,
		})
	}, &submitResp)
	if err != nil {
		return 0, err
	}
	if len(submitResp.Data.TaskList) == 0 {
		return 0, fmt.Errorf("offline submit failed: empty task list")
	}
	if submitResp.Data.TaskList[0].Result != 0 {
		return 0, fmt.Errorf("offline submit failed")
	}
	if submitResp.Data.TaskList[0].TaskID == 0 {
		return 0, fmt.Errorf("offline submit failed: empty task id")
	}
	return submitResp.Data.TaskList[0].TaskID, nil
}

func (d *Pan123) GetOfflineTask(ctx context.Context, taskID int64) (*offlineTask, error) {
	if taskID == 0 {
		return nil, fmt.Errorf("invalid task id")
	}
	page := 1
	pageSize := 100
	statusArr := []int{0, 1, 2, 3}
	for {
		var listResp offlineTaskListResp
		_, err := d.Request(OfflineTaskList, http.MethodPost, func(req *resty.Request) {
			req.SetContext(ctx).SetBody(base.Json{
				"current_page": page,
				"page_size":    pageSize,
				"status_arr":   statusArr,
			})
		}, &listResp)
		if err != nil {
			return nil, err
		}
		for i := range listResp.Data.List {
			if listResp.Data.List[i].TaskID == taskID {
				return &listResp.Data.List[i], nil
			}
		}
		if len(listResp.Data.List) == 0 || page*pageSize >= listResp.Data.Total {
			break
		}
		page++
	}
	return nil, ErrOfflineTaskNotFound
}

func (d *Pan123) DeleteOfflineTasks(ctx context.Context, taskIDs []int64) error {
	if len(taskIDs) == 0 {
		return nil
	}
	_, err := d.Request(OfflineTaskDelete, http.MethodPost, func(req *resty.Request) {
		req.SetContext(ctx).SetBody(base.Json{
			"task_ids": taskIDs,
		})
	}, nil)
	return err
}

func (d *Pan123) getFiles(ctx context.Context, parentId string, name string) ([]File, error) {
	page := 1
	total := 0
	res := make([]File, 0)
	// 2024-02-06 fix concurrency by 123pan
	for {
		if err := d.APIRateLimit(ctx, FileList); err != nil {
			return nil, err
		}
		var resp Files
		query := map[string]string{
			"driveId":              "0",
			"limit":                "100",
			"next":                 "0",
			"orderBy":              "file_id",
			"orderDirection":       "desc",
			"parentFileId":         parentId,
			"trashed":              "false",
			"SearchData":           "",
			"Page":                 strconv.Itoa(page),
			"OnlyLookAbnormalFile": "0",
			"event":                "homeListFile",
			"operateType":          "4",
			"inDirectSpace":        "false",
		}
		_res, err := d.Request(FileList, http.MethodGet, func(req *resty.Request) {
			req.SetQueryParams(query)
		}, &resp)
		if err != nil {
			return nil, err
		}
		log.Debug(string(_res))
		page++
		res = append(res, resp.Data.InfoList...)
		total = resp.Data.Total
		if len(resp.Data.InfoList) == 0 || resp.Data.Next == "-1" {
			break
		}
	}
	if len(res) != total {
		log.Warnf("incorrect file count from remote at %s: expected %d, got %d", name, total, len(res))
	}
	return res, nil
}

func (d *Pan123) getUserInfo(ctx context.Context) (*UserInfoResp, error) {
	var resp UserInfoResp
	_, err := d.Request(UserInfo, http.MethodGet, func(req *resty.Request) {
		req.SetContext(ctx)
	}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
