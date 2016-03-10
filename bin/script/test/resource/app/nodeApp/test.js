var url = require('url');
var http = require('http');
//var appmetrics = require('appmetrics');
//var monitor = appmetrics.monitor();
var arr1 = [];
var arr2 = [];
var arr3 = [];
var arr4 = [];
var arr5 = [];
var arr6 = [];
var arr7 = [];
var arr8 = [];
var arr9 = [];
var arr0 = [];
var flag = 1;
var maxMem = 500; // mb
/*monitor.on('gc', function (gc) {
	
	var processMem = process.memoryUsage();
    console.log('--->[' + new Date(gc.time) + '] GC: heap: type' + gc.type +', heapSize:'+(gc.size/(1024*1024)).toFixed(2)+'m, usedHeap:'+(gc.used/(1024*1024)).toFixed(2)+'m, duration:'+gc.duration/(1000));
	console.log('---->HEAP2] total: ' +  (processMem.heapTotal/(1024*1024)).toFixed(2) + ' used:' + (processMem.heapUsed/(1024*1024)).toFixed(2));
});*/
setInterval(function(){
	var mem1 = process.memoryUsage();
	console.log('---mem used:' + (mem1.rss/(1024*1024)).toFixed(2) + 'm');
	console.log('---heap total:' + (mem1.heapTotal/(1024*1024)).toFixed(2) + 'm');
	console.log('---heap used:' + (mem1.heapUsed/(1024*1024)).toFixed(2) + 'm');
},5000);
var server = http.createServer(function handler(req, res) {
	var params = url.parse(req.url, true).query;
	var cmd = params.cmd;
	var cmdType= params.mode;
	if('add' == cmd){
			var mem = process.memoryUsage();
			if(mem.rss/(1024*1024) > maxMem ){	
			}
			else{
					//console.log('mem < 500m-------------------');
					var num = params.num;
					for(var i = 0; i < num; i++){						
							arr0.push(Math.random());
							arr1.push(Math.random());
							arr2.push(Math.random());
							arr3.push(Math.random());
							arr4.push(Math.random());
							arr5.push(Math.random());
							arr6.push(Math.random());
							arr7.push(Math.random());
							arr8.push(Math.random());
							arr9.push(Math.random());
						}
			}
			
			res.end('the length of array is ' + arr0.length );
		
		
	}
	else if('remove' == cmd){
		var num = params.num;
		//console.log('remove request, it will remove ' + num + 'elements from the array');
		if(num > arr0.length){
			num = arr0.length;
		}
		arr0.length = arr0.length - num;
		arr1.length = arr1.length - num;
		arr2.length = arr2.length - num;
		arr3.length = arr3.length - num;
		arr4.length = arr4.length - num;
		arr5.length = arr5.length - num;
		arr6.length = arr6.length - num;
		arr7.length = arr7.length - num;
		arr8.length = arr8.length - num;
		arr9.length = arr9.length - num;
		//console.log('remove request, after removing the size of array is ' + arr.length);
		res.end('the length of array is ' + arr0.length );
	}
	else if('destroy' == cmd){
		arr0.length = 0;
		arr1.length = 0;
		arr2.length = 0;
		arr3.length = 0;
		arr4.length = 0;
		arr5.length = 0;
		arr6.length = 0;
		arr7.length = 0;
		arr8.length = 0;
		arr9.length = 0;
		console.log('destroy request, the array size is now 0');
		res.end('the length of array is ' + arr0.length );
	}
	else if('print' == cmd){
		console.log('--------------------------->>>>>>>>>>>>>>>');
		var mem = process.memoryUsage();
		console.log('mem used:' + (mem.rss/(1024*1024)).toFixed(2) + 'm');
		console.log('heap total:' + (mem.heapTotal/(1024*1024)).toFixed(2) + 'm');
		console.log('heap used:' + (mem.heapUsed/(1024*1024)).toFixed(2) + 'm');
		console.log('arr size is ' + arr0.length);
		res.end('the length of array is ' + arr0.length);
		
	}
}).listen(process.env.PORT || 3000);

console.log('App listening on port 3000');
