{ bundlerApp }:

bundlerApp {
  pname = "cf-uaac";
  gemdir = ./.;
  exes = ["uaac"];

  meta = {
    description = "CloudFoundry UAA Command Line Client";
    homepage = "https://github.com/cloudfoundry/cf-uaac";
    mainProgram = "uaac";
  };
}
