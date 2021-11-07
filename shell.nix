#!/usr/bin/env nix-shell
#! nix-shell --pure --run "env i_fcolor=red zsh" .

let
     pkgs = import (builtins.fetchGit {
         # Descriptive name to make the store path easier to identify
         name = "my-old-revision";
         url = "https://github.com/NixOS/nixpkgs/";
         ref = "refs/heads/nixpkgs-unstable";
         rev = "860b56be91fb874d48e23a950815969a7b832fbc";
     }) {};

     goPkg = pkgs.go;
in with pkgs;

pkgs.mkShell {
  buildInputs = with pkgs; [
      go
      gopls
      goimports
      darwin.apple_sdk.frameworks.CoreFoundation
      darwin.apple_sdk.frameworks.Foundation
      darwin.apple_sdk.frameworks.LocalAuthentication

      # keep this line if you use bash
      pkgs.bashInteractive
    ];

  shellHook = ''
    unset GOPATH GOROOT
    export NIX_LDFLAGS="-F${pkgs.darwin.apple_sdk.frameworks.CoreFoundation}/Library/Frameworks -framework CoreFoundation $NIX_LDFLAGS";
  '';
}
