'use strict'

var path = require("path");
var expect = require("chai").expect;
var helper = require(path.join(__dirname, "../../../lib/routes/routeHelper.js"));

describe("routeHelper", function() {
  describe("isMissing", function() {
    var toCheckObj;
    beforeEach(function() {
      toCheckObj = { value: "anything" };
    });
    context("check valid value", function() {
      it("return false", function() {
        expect(helper.isMissing(toCheckObj.value)).to.equal(false);
      });
    });
    context("check null", function() {
      it("return true", function() {
        toCheckObj.value = null;
        expect(helper.isMissing(toCheckObj.value)).to.equal(true);
      });
    });
    context("check undefined", function() {
      it("return true", function() {
        delete toCheckObj.value;
        expect(helper.isMissing(toCheckObj.value)).to.equal(true);
      });
    });

  });

  describe("isInteger", function() {
    var intVar;
    beforeEach(function() {
      intVar = 1;
    });
    context("check integer", function() {
      it("return true", function() {
        expect(helper.isInteger(intVar)).to.equal(true);
      });
    });
    context("check string", function() {
      it("return false", function() {
        intVar = "not-integer-string";
        expect(helper.isInteger(intVar)).to.equal(false);
      });
    });
    context("check array", function() {
      it("return false", function() {
        intVar = [];
        expect(helper.isInteger(intVar)).to.equal(false);
      });
    });
    context("check obj", function() {
      it("return false", function() {
        intVar = {};
        expect(helper.isInteger(intVar)).to.equal(false);
      });
    });

  });

  describe("isString", function() {
    var strVar;
    beforeEach(function() {
      strVar = "string";
    });
    context("check string", function() {
      it("return true", function() {
        expect(helper.isString(strVar)).to.equal(true);
      });
    });
    context("check number", function() {
      it("return false", function() {
        strVar = 123;
        expect(helper.isString(strVar)).to.equal(false);
      });
    });
    context("check array", function() {
      it("return false", function() {
        strVar = [];
        expect(helper.isInteger(strVar)).to.equal(false);
      });
    });
    context("check obj", function() {
      it("return false", function() {
        strVar = {};
        expect(helper.isInteger(strVar)).to.equal(false);
      });
    });

  });

  describe("pagination", function() {
    var resultList = [];
    var result = {};
    context("paginate the resultList", function() {
      resultList = [
        { 'app_id': "theAppId", 'timestamp': 300, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 250, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 200, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 150, 'scaling_type': 1, 'status': 1, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' },
        { 'app_id': "theAppId", 'timestamp': 100, 'scaling_type': 0, 'status': 0, 'old_instances': 2, 'new_instances': 4, 'reason': 'a reason', 'message': '', 'error': '' }
      ];
      it("get the 1st page", function() {
        result = helper.pagination(resultList, 1, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 1,
          resources: resultList.slice(0, 2)
        });
      });

      it("get the 2nd page", function() {
        result = helper.pagination(resultList, 2, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 2,
          resources: resultList.slice(2, 4)
        });
      });

      it("get the 3rd page and only has one record", function() {
        result = helper.pagination(resultList, 3, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 3,
          resources: resultList.slice(4)
        });
      });

      it("get the 4th page and there is no record", function() {
        result = helper.pagination(resultList, 4, 2);
        expect(result).to.deep.equal({
          total_results: 5,
          total_pages: 3,
          page: 4,
          resources: []
        });
      });

    });
  });
});
