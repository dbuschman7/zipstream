{
    pkgs ? import <nixpkgs> { },
}:

pkgs.mkShell {
    name = "dev-env";
    buildInputs = [
        pkgs.go
    ];
    shellHook = ''
       # bash scripts here
       echo "Env started ..."
    '';
}
