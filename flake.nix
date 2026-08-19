{
  description = "A keyboard-driven launcher for the terminal";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        rec {
          launtui = pkgs.buildGoModule {
            pname = "launtui";

            version = builtins.head (
              builtins.match ".*const version = \"([^\"]+)\".*" (builtins.readFile ./main.go)
            );

            src = ./.;

            vendorHash = "sha256-4EomlPmKRU1Ptq1oHicZo8MtUBYAqgx6AI6Xy3aG/fo=";

            env.CGO_ENABLED = 0;

            ldflags = [
              "-s"
              "-w"
            ];

            postInstall = ''
              install -Dm644 contrib/systemd/launtui-watch.service \
                $out/lib/systemd/user/launtui-watch.service

              substituteInPlace $out/lib/systemd/user/launtui-watch.service \
                --replace-fail /usr/local/bin/launtui $out/bin/launtui

              install -Dm644 contrib/desktop/launtui.desktop \
                $out/share/applications/launtui.desktop
            '';

            meta = {
              description = "A keyboard-driven launcher for the terminal";
              homepage = "https://github.com/siliconwitch/launtui";
              license = nixpkgs.lib.licenses.mit;
              mainProgram = "launtui";
              platforms = nixpkgs.lib.platforms.linux;
            };
          };

          default = launtui;
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.goreleaser
            ];
          };
        }
      );

      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.services.launtui;
        in
        {
          options.services.launtui = {
            enable = lib.mkEnableOption "launtui, with the clipboard watcher running in the background";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.stdenv.hostPlatform.system}.launtui;
              defaultText = lib.literalExpression "launtui";
              description = "The launtui package to install and run the clipboard watcher from.";
            };
          };

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];

            systemd.user.services.launtui-watch = {
              description = "launtui clipboard watcher";

              wantedBy = [ "graphical-session.target" ];
              partOf = [ "graphical-session.target" ];
              after = [ "graphical-session.target" ];

              serviceConfig = {
                ExecStart = "${lib.getExe cfg.package} -watch";
                Restart = "on-failure";
                RestartSec = 10;
              };
            };
          };
        };
    };
}
