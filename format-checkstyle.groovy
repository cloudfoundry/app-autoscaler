#!/usr/bin/env groovy

import groovy.xml.XmlParser

/*
<file name="/Users/C5326931/workspace/app-autoscaler/scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/rest/model/ApplicationSchedules.java">
<error line="5" column="1" severity="warning" message="Extra separation in import group before &apos;io.swagger.annotations.ApiModel&apos;" source="com.puppycrawl.tools.checkstyle.checks.imports.CustomImportOrderCheck"/>
<error line="8" severity="warning" message="Summary javadoc is missing." source="com.puppycrawl.tools.checkstyle.checks.javadoc.SummaryJavadocCheck"/>
<error line="14" column="1" severity="warning" message="Line contains a tab character." source="com.puppycrawl.tools.checkstyle.checks.whitespace.FileTabCharacterCheck"/>
<error line="14" column="9" severity="warning" message="&apos;member def modifier&apos; has incorrect indentation level 8, expected level should be 2." source="com.puppycrawl.tools.checkstyle.checks.indentation.IndentationCheck"/>
<error line="15" column="1" severity="warning" message="Line contains a tab character." source="com.puppycrawl.tools.checkstyle.checks.whitespace.FileTabCharacterCheck"/>
<error line="16" column="1" severity="warning" message="Line contains a tab character." source="com.puppycrawl.tools.checkstyle.checks.whitespace.FileTabCharacterCheck"/>
<error line="18" column="1" severity="warning" message="Line contains a tab character." source="com.puppycrawl.tools.checkstyle.checks.whitespace.FileTabCharacterCheck"/>
<error line="18" column="9" severity="warning" message="&apos;member def modifier&apos; has incorrect indentation level 8, expected level should be 2." source="com.puppycrawl.tools.checkstyle.checks.indentation.IndentationCheck"/>
*/
def results = new XmlParser().parse(new File('scheduler/target/checkstyle-result.xml'))
results.file.each{ file ->
  file.error.each { error ->
    println "::${error.@severity} file=${file.@name},line=${error.@line},col=${error.@column}::${error.@message}"
  }
}
