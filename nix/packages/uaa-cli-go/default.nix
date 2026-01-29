{ lib, stdenv, fetchurl }:

let
  version = "0.17.0";

  sources = {
    x86_64-linux = {
      url = "https://github.com/cloudfoundry/uaa-cli/releases/download/${version}/uaa-linux-amd64-${version}";
      sha256 = "27127fb3604d26e1c17fe8a62b08e6e0c33d4b67a35f12c8c20b6cb7e604a70f";
    };
    aarch64-linux = {
      url = "https://github.com/cloudfoundry/uaa-cli/releases/download/${version}/uaa-linux-amd64-${version}";
      sha256 = "27127fb3604d26e1c17fe8a62b08e6e0c33d4b67a35f12c8c20b6cb7e604a70f";
    };
    x86_64-darwin = {
      url = "https://github.com/cloudfoundry/uaa-cli/releases/download/${version}/uaa-darwin-amd64-${version}";
      sha256 = "0d3303bea01ab033199b1da1f6ac223f10c0bb370a1b1cec25362ef4468e52e5";
    };
    aarch64-darwin = {
      url = "https://github.com/cloudfoundry/uaa-cli/releases/download/${version}/uaa-darwin-arm64-${version}";
      sha256 = "79099aa3ad4369202e318d50f49a9282b2f0b4c2b8bbfb45e323a65a3c52bfef";
    };
  };

  source = sources.${stdenv.system} or (throw "Unsupported system: ${stdenv.system}");

in stdenv.mkDerivation {
  pname = "uaa-cli";
  inherit version;

  src = fetchurl {
    inherit (source) url sha256;
  };

  dontUnpack = true;
  dontBuild = true;

  installPhase = ''
    install -D -m755 $src $out/bin/uaa
  '';

  meta = with lib; {
    description = "CloudFoundry UAA Command Line Client (Go version)";
    homepage = "https://github.com/cloudfoundry/uaa-cli";
    license = licenses.asl20;
    mainProgram = "uaa";
    platforms = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
  };
}
