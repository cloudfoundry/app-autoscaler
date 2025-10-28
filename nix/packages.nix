{ buildGoModule
, callPackage
, fetchFromGitHub
, fetchgit
, lib
}: {
  app-autoscaler-cli-plugin = buildGoModule rec {
    pname = "app-autoscaler-cli-plugin";

    major = "4";
    minor = "1";
    patch = "2";
    version = "${major}.${minor}.${patch}";
    src = fetchgit {
      url = "https://github.com/cloudfoundry/app-autoscaler-cli-plugin";
      rev = "refs/tags/v${version}";
      hash = "sha256-WsX6TY0SOO1HMBfmBlZ6KPeWlLzPJh4t5hmn5aNWPzk=";
      fetchSubmodules = true;
    };
    vendorHash = "sha256-yLI4gYciEqH1vmT4ILuY1gYCm6ehCjh7dazcEib+vkY=";

    ldflags = ["-s" "-w"
      "-X 'main.BuildMajorVersion=${major}'"
      "-X 'main.BuildMinorVersion=${minor}'"
      "-X 'main.BuildPatchVersion=${patch}'"
    ];

    doCheck = false;

    meta = {
      longDescription = ''
      App-AutoScaler plug-in provides the command line interface to manage
      [App AutoScaler](<https://github.com/cloudfoundry/app-autoscaler-release>)
      policies, retrieve metrics and scaling event history.
      '';
      homepage = "https://github.com/cloudfoundry/app-autoscaler-cli-plugin";
      license = [lib.licenses.asl20];
    };
  };

  # This bosh-bootloader custom build can be removed once
  # <https://github.com/cloudfoundry/bosh-bootloader/issues/596> is implemented. Code inspired by
  # <https://github.com/cloudfoundry/bosh-bootloader/issues/596#issuecomment-1959853091>.
  bosh-bootloader = buildGoModule rec {
    pname = "bosh-bootloader";
    version = "9.0.34";
    src = fetchgit {
      url = "https://github.com/cloudfoundry/bosh-bootloader";
      rev = "refs/tags/v${version}";
      fetchSubmodules = true;
      hash = "sha256-14swFtRjbN45MyRu5k9HYZiOJRxgtS82roWo6XCJx3U=";
    };
    vendorHash = null;

    ldflags = [
      "-X main.Version=v${version}"
    ];

    doCheck = false;

    subPackages = ["bbl"];

    meta = with lib; {
      description = "Command line utility for standing up a BOSH director on an IAAS of your choice.";
      homepage = "https://github.com/cloudfoundry/bosh-bootloader";
      license = licenses.asl20;
    };
  };

  cloud-mta-build-tool = buildGoModule rec {
    pname = "Cloud MTA Build Tool";
    version = "1.2.30";

    src = fetchFromGitHub {
      owner = "SAP";
      repo = "cloud-mta-build-tool";
      rev = "refs/tags/v${version}";
      hash = "sha256-iuNaaApnyfyqm3SvYG3en+a78MUP1BxSM3JZz+JhEFs=";
    };
    vendorHash = "sha256-pyXeuZGg3Yv6p8GNKC598EdZqX8KLc3rkewMkq4vA7c=";

    ldflags = ["-s" "-w" "-X main.Version=${version}"];

    doCheck = false;

    postInstall = ''
      pushd "''${out}/bin" &> /dev/null
        ln --symbolic 'cloud-mta-build-tool' 'mbt'
      popd
    '';

    meta = with lib; {
      description = "Multi-Target Application (MTA) build tool for Cloud Applications";
      homepage = "https://sap.github.io/cloud-mta-build-tool";
      license = licenses.asl20;
    };
  };

  log-cache-cli-plugin = buildGoModule rec {
    pname = "log-cache-cli";
    version = "6.2.1";
    src = fetchgit {
      url = "https://github.com/cloudfoundry/log-cache-cli";
      rev = "refs/tags/v${version}";
      hash = "sha256-A7DbmwZ20oBouH7ArxSSXertlzeMnCL814+jfyPiGCQ=";
      fetchSubmodules = true;
    };
    vendorHash = null;

    ldflags = ["-s" "-w" "-X main.version=${version}"];

    doCheck = false;

    meta = with lib; {
      description = "A cf CLI plugin for interacting with Log Cache.";
      homepage = "https://github.com/cloudfoundry/log-cache-cli";
      license = licenses.asl20;
    };
  };

  cf-deploy-plugin = buildGoModule rec {
    pname = "CF Deploy Plugin";
    version = "3.5.0";

    src = fetchFromGitHub {
      owner = "cloudfoundry";
      repo = "multiapps-cli-plugin";
      rev = "v${version}";
      hash = "sha256-SVPVPJJWOk08ivZWu9UwD9sIISajIukQpcFpc0tU1zg=";
    };
    vendorHash = "sha256-S066sNHhKxL4anH5qSSBngtOcAswopiYBXgKAvHyfAM=";

    env.CGO_ENABLED = 0;
    ldflags = ["-w -X main.Version=${version}"];

    meta = with lib; {
      description = "";
      longDescription = ''
        This is a Cloud Foundry CLI plugin (formerly known as CF MTA Plugin) for performing
        operations on
        [Multitarget Applications (MTAs)](<https://www.sap.com/documents/2021/09/66d96898-fa7d-0010-bca6-c68f7e60039b.html>)
        in Cloud Foundry, such as deploying, removing, viewing, etc. It is a client for the
        [CF MultiApps Controller](<https://github.com/cloudfoundry-incubator/multiapps-controller>)
        (known also as CF MTA Deploy Service), which is an MTA deployer implementation for Cloud
        Foundry. The business logic and actual processing of MTAs happens into
        CF MultiApps Controller backend.
      '';
      homepage = "https://github.com/cloudfoundry/multiapps-cli-plugin";
      license = licenses.asl20;
    };
  };

  uaac = callPackage ./packages/uaac {};
}
