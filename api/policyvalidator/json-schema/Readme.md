# Read-me #

The currently used golang-library for json-schema, namely “[gojsonschema](<https://github.com/xeipuuv/gojsonschema>)” not seems to be capable to resolve references to other files correctly across different directories. Perhaps not limited to but especially when referencing up the file-system-hierarchy (parent-directories). In theory, symbolic links can be used to circumvent the issue … if there would not be another weakness in <https://github.com/cloudfoundry/bosh-compile-action> which we currently use in the workflow [bosh-release-checks.yaml](<../../../../.github/workflows/bosh-release-checks.yaml>) for making a compiled release: It can not handle symbolic links. Therefore it gets hard-linked here (which can not be tracked by git but it can somehow handle it!).

After phasing out the bosh-technology of “Application Autoscaler”, these hardlinks can be removed.

⚠️ This means the consinstence needs to be ensured manually. There is no CI/CD-check because Bosh is already in its phase-out for “Application Autoscaler”.
