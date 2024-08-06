{
  description = "TwitchMenu program to see who is live";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }@inputs:
    let inherit (flake-utils.lib) eachDefaultSystem mkApp;
    in eachDefaultSystem (system:
        let pkgs = import nixpkgs { inherit system; };
        in {
          apps.default = mkApp {
            drv = pkgs.buildGoModule rec {
              pname = "twitchmenu";
              version = "1.0";

              src = ./.;
              vendorHash = null;

              ldflags = [
                "-s -w -X github.com/aiten/twitchmenu/cmd.version=${version}"
              ];

              nativeBuildInputs = [pkgs.musl];

              CGO_ENABLED = 0;

              env = {
                TWITCH_API_KEY  = builtins.getEnv "TWITCH_API_KEY";
                TWITCH_API_SECRET  = builtins.getEnv "TWITCH_API_SECRET";
              };

              meta = with pkgs.lib; {
                description = "dmenu script to see who is online";
                homepage = "https://github.com/aiten/twitchmenu";
                license = licenses.gpl3;
                maintainers = with maintainers; [ ait ];
              };
            };
            exePath = "/bin/twitchmenu";
        };
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [ go libcap gcc];
        };
      });
}
