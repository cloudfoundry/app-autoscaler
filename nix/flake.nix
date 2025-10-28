{
  description = "Extra dependencies of app-autoscaler-release for the devbox";

  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/nixos-unstable;
  };

  outputs = { self, nixpkgs}:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in {
      packages = forAllSystems (system:
        let
          nixpkgs = nixpkgsFor.${system};
          callPackages = nixpkgs.lib.customisation.callPackagesWith nixpkgs;
        in callPackages ./packages.nix {}
      );

      # 🚸 Having flake.nix on top-level makes this definition easily consumable with Nix, e.g. for
      # other repositories of Autoscaler. ⚠️ This currently is for internal use only. Autoscaler does
      # not officially support this flake-output.
      openapi-specifications = {
        app-autoscaler-api =
          let apiPath = ./api;
          in builtins.filterSource
            (path: type: builtins.match ".*\.ya?ml" (baseNameOf path) != null && type == "regular")
            apiPath;
      };
  };
}
