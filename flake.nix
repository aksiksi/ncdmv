{
  description = "Find upcoming NC DMV appointments";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs, ... }: let
    # No chromium on darwin: https://github.com/NixOS/nixpkgs/issues/247855
    supportedSystems = [ "x86_64-linux" "aarch64-linux" ];
    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    pname = "ncdmv";
    owner = "aksiksi";
    version = "0.1.0";
  in {
    packages = forAllSystems (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = nixpkgs.lib;
      in {
        default = pkgs.buildGoModule {
          inherit pname;
          inherit version;
          src = ./.;
          vendorHash = "sha256-U0HNh5MrR1ZftfUC2Z7jPc7Me6OM/mj2bGvySLrBUus=";
          buildInputs = [
            pkgs.sqlite
          ];

          # We want Chrome to be available in $PATH when executing the server.
          # The way to do this is to wrap the binary into a thin shell script
          # that adds the chromium Nix store path to $PATH before executing
          # the binary.
          #
          # See: https://discourse.nixos.org/t/buildinputs-not-propagating-to-the-derivation/4975/2
          nativeBuildInputs = [ pkgs.makeWrapper ];
          wrapperPath = lib.makeBinPath ([ pkgs.chromium ]);
          postFixup = let
            wrapperPath = self.outputs.packages.${system}.default.wrapperPath; in ''
            wrapProgram $out/bin/server \
                --prefix PATH : "${wrapperPath}"
          '';
        };
        docker = let
          packagePath = self.outputs.packages.${system}.default;
            in pkgs.dockerTools.buildImage {
          name = "ghcr.io/aksiksi/ncdmv";
          tag = "latest";
          config.Cmd = [ "${packagePath}/bin/server" ];
        };
      }
    );

    # Development shell
    # nix develop
    devShells = forAllSystems (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        default = pkgs.mkShell {
          buildInputs = [
            pkgs.chromium
            pkgs.sqlite
          ];
          packages = [
            pkgs.go
            pkgs.gopls
          ];
        };
      }
    );
  };
}

