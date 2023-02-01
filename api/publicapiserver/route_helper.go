package publicapiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

const (
	ASC  = "ASC"
	DESC = "DESC"
)

func parseParameter(r *http.Request, vars map[string]string) (*url.Values, error) {
	appId := vars["appId"]
	startTime := r.URL.Query().Get("start-time")
	endTime := r.URL.Query().Get("end-time")
	orderDirection := r.URL.Query().Get("order-direction")
	order := r.URL.Query().Get("order")
	page := r.URL.Query().Get("page")
	resultsPerPage := r.URL.Query().Get("results-per-page")

	if appId == "" {
		return nil, fmt.Errorf("appId is required")
	}

	if startTime == "" {
		startTime = "0"
	}
	_, err := strconv.Atoi(startTime)
	if err != nil {
		return nil, fmt.Errorf("start-time must be an integer")
	}

	if endTime == "" {
		endTime = "-1"
	}
	_, err = strconv.Atoi(endTime)
	if err != nil {
		return nil, fmt.Errorf("end-time must be an integer")
	}

	if orderDirection == "" && order == "" {
		orderDirection = DESC
	} else if orderDirection == "" && order != "" {
		orderDirection = order
	}
	orderDirection = strings.ToUpper(orderDirection)
	if orderDirection != DESC && orderDirection != ASC {
		return nil, fmt.Errorf("order-direction must be DESC or ASC")
	}
	if page == "" {
		page = "1"
	}
	pageNo, err := strconv.Atoi(page)
	if err != nil {
		return nil, fmt.Errorf("page must be an integer")
	}
	if pageNo <= 0 {
		return nil, fmt.Errorf("page must be greater than 0")
	}

	if resultsPerPage == "" {
		resultsPerPage = "50"
	}
	resultsPerPageCount, err := strconv.Atoi(resultsPerPage)
	if err != nil {
		return nil, fmt.Errorf("results-per-page must be an integer")
	}
	if resultsPerPageCount <= 0 {
		return nil, fmt.Errorf("results-per-page must be greater than 0")
	}
	parameters := &url.Values{}
	parameters.Add("start", startTime)
	parameters.Add("end", endTime)
	parameters.Add("order", orderDirection)
	parameters.Add("page", page)
	parameters.Add("results-per-page", resultsPerPage)

	return parameters, nil
}

func paginateResource(resourceList []byte, parameters *url.Values, r *url.URL) (interface{}, error) {
	var resourceListItems []interface{}

	err := json.Unmarshal(resourceList, &resourceListItems)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources")
	}
	totalResults := len(resourceListItems)
	perPage, _ := strconv.Atoi(parameters.Get("results-per-page"))
	pageNo, _ := strconv.Atoi(parameters.Get("page"))

	totalPages := 0
	if (totalResults % perPage) == 0 {
		totalPages = totalResults / perPage
	} else {
		totalPages = totalResults/perPage + 1
	}

	startIndex := (pageNo - 1) * perPage
	if startIndex > totalResults {
		startIndex = totalResults
	}
	endIndex := startIndex + perPage
	if endIndex > totalResults {
		endIndex = totalResults
	}

	resources := resourceListItems[startIndex:endIndex]
	prevUrl := ""
	if (pageNo > 1) && (pageNo <= totalPages+1) {
		prevUrl = getPageUrl(r, pageNo-1)
	}

	nextUrl := ""
	if pageNo < totalPages {
		nextUrl = getPageUrl(r, pageNo+1)
	}

	result := models.PublicApiResponseBase{}

	result.TotalResults = totalResults
	result.TotalPages = totalPages
	result.Page = pageNo
	result.PrevUrl = prevUrl
	result.NextUrl = nextUrl
	result.Resources = resources

	return result, nil
}

func getPageUrl(r *url.URL, targetPageNo int) string {
	pageUrl := *r
	queries := pageUrl.Query()
	pageParams := url.Values{}
	for key, value := range queries {
		if key == "page" {
			pageParams.Add(key, strconv.Itoa(targetPageNo))
		} else {
			pageParams.Add(key, value[0])
		}
	}

	pageUrl.RawQuery = pageParams.Encode()
	return pageUrl.String()
}
