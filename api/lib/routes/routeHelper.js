'use strict';

exports.DESC = "DESC";
exports.ASC = "ASC";

exports.isMissing = function(value) {
  return typeof(value) === "undefined" || value === null;
}
exports.isInteger = function(value) {
  return Number.isInteger(value);
}
exports.isString = function(value) {
  return typeof(value) === "string";
}

exports.pagination = function(resultList, page, resultsPerPage, req) {
  var result = {};
  var totalResults = resultList.length;
  var totalPages = 0;
  var perPage = Number.parseInt(resultsPerPage);
  var pageNo = Number.parseInt(page);
  var prevUrl = null;
  var nextUrl = null;
  totalPages = (totalResults % perPage) === 0 ? (totalResults / perPage) : (Number.parseInt(totalResults / perPage) + 1);
  var startIndex = (pageNo - 1) * perPage;
  var endIndex = startIndex + perPage;
  var resources = resultList.slice(startIndex, endIndex);
  var originalUrl = req.originalUrl
  if (page > 1 && page <= totalPages + 1) {
    var prevPage = page - 1;
    prevUrl = originalUrl.replace("?page=" + page, "?page=" + prevPage);
    prevUrl = prevUrl.replace("&page=" + page, "&page=" + prevPage);
  }
  if (page < totalPages) {
    var nextPage = page + 1;
    nextUrl = originalUrl.replace("?page=" + page, "?page=" + nextPage);
    nextUrl = nextUrl.replace("&page=" + page, "&page=" + nextPage);
  }
  result.total_results = totalResults;
  result.total_pages = Number.parseInt(totalPages);
  result.page = Number.parseInt(pageNo);
  result.prev_url = prevUrl;
  result.next_url = nextUrl;
  result.resources = resources;
  return result;
}