{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }: 
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        package = pkgs.buildGoModule {
          pname = "foxingress";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-z9cn+KnorcSAJRea2sNHBv4zDxVeZ5Eoi7j7A9yV5Mg=";
          buildInputs = [];
        };
      in
      {
        packages = {
          default = package;
          foxingress = package;
        };
      });
}
