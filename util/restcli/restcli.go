package restcli

import "resty.dev/v3"

type Params struct {
	Header  map[string]string
	Queries map[string]string
	Body    any

	Formdata map[string]string

	Files map[string]string // key is the name of the file, value is the path to the file
}

func Get(cli *resty.Client, url string, params *Params, result ...any) (*resty.Response, error) {
	req := genReq(cli, params, result)
	return req.Get(url)
}

func Post(cli *resty.Client, url string, params *Params, result ...any) (*resty.Response, error) {
	req := genReq(cli, params, result)
	return req.Post(url)
}

func Put(cli *resty.Client, url string, params *Params, result ...any) (*resty.Response, error) {
	req := genReq(cli, params, result)
	return req.Put(url)
}

func Patch(cli *resty.Client, url string, params *Params, result ...any) (*resty.Response, error) {
	req := genReq(cli, params, result)
	return req.Patch(url)
}

func Delete(cli *resty.Client, url string, params *Params, result ...any) (*resty.Response, error) {
	req := genReq(cli, params, result)
	return req.Delete(url)
}

func genReq(cli *resty.Client, params *Params, result []any) *resty.Request {
	req := cli.R()

	if len(result) != 0 {
		req = req.SetResult(result[0])
	}

	if params == nil {
		return req
	}

	if params.Header != nil {
		req = req.SetHeaders(params.Header)
		if _, ok := params.Header["Accept"]; !ok || params.Header["Accept"] == "" {
			req = req.SetHeader("Accept", "application/json")
		}
	} else {
		req = req.SetHeader("Accept", "application/json")
	}

	if params.Queries != nil {
		req = req.SetQueryParams(params.Queries)
	}

	if params.Body != nil {
		req = req.SetBody(params.Body)
		if params.Header == nil || params.Header["Content-Type"] == "" {
			req = req.SetHeader("Content-Type", "application/json")
		}
	}

	if params.Files != nil {
		req = req.SetFiles(params.Files)
	}

	if params.Formdata != nil {
		req = req.SetFormData(params.Formdata)
	}
	return req
}
