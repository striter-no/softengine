#!/usr/bin/python
import os, sys

if __name__ == "__main__":
    path = sys.argv[1]
    if path.startswith('./'): path = path[2:]
    if path == "":
        path = "."

    name = f"./.dev/{path.replace('/', '_') if path != "." else "repo-dump"}.txt"
    gitign = "./.dev/.gitignore"

    # parser = argparse.ArgumentParser()
    # parser.add_argument("-d", "--dir", required=True, help="Directory with code to dump")
    # parser.add_argument("-n", "--name", required=True, help="Name.ext of the dump file")
    # parser.add_argument("-g", "--git_ignore", required=False, help="If set, ignore files/directories in given .gitignore")
    # args = parser.parse_args()

    if os.path.exists(name):
        os.remove(name)

    # if gitign:
    with open(gitign) as f:
        ignore = [l for l in f.read().split('\n') if len(l) > 0]
    # else:
    #     ignore = []

    directories = ""
    dumped_text = ""
    for root, dirs, files in os.walk(path):
        print(f"[+] {root} ({len(files)} files)")
        directories += f"- {root}\n"

        dirs[:] = [d for d in dirs if d not in ignore]
        for file in files:
            if file in ignore:
                continue

            fr = file.rsplit('.', 1)
            if len(fr) >= 2 and "*." + fr[1] in ignore:
                continue

            path = os.path.join(root, file)
            with open(path) as f:
                dumped_text += f"`{path}`:\n```\n{"\n".join([f' {i + 1} | {l}' for i, l in enumerate(f.read().split('\n'))])}\n```\n\n"

    dumped_text = f"directories:\n{directories}" + '\n\n\n' + dumped_text
    with open(name, 'w') as f:
        f.write(dumped_text)
