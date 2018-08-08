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

exports.pagination = function(resultList, page, resultsPerPage) {
  var result = {};
  var totalResults = resultList.length;
  var totalPages = 0;
  var perPage = Number.parseInt(resultsPerPage);
  var pageNo = Number.parseInt(page);
  totalPages = (totalResults % perPage) === 0 ? (totalResults / perPage) : (totalResults / perPage + 1);
  var startIndex = (pageNo - 1) * perPage;
  var endIndex = startIndex + perPage;
  var resources = resultList.slice(startIndex, endIndex);
  result.total_results = totalResults;
  result.total_pages = Number.parseInt(totalPages);
  result.page = Number.parseInt(pageNo);
  result.resources = resources;
  return result;
}
