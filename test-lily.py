#!/usr/bin/python

import getopt
import os
import sys
import re
import urllib.request
import json

class Options:
    def __init__(self):
        self.platforms=set([])
        self.stage=""
        self.mode=""


def error(msg):
    sys.stderr.write("%s\n" % msg)
    sys.exit(1)

def system(cmd):
    print(cmd)
    stat =  os.system(cmd)
    assert stat == 0, stat

def usage():
    sys.stdout.write("""
test-lily [options] URL BRANCH
test-lily [options] rietveld CHANGE-NUM

options:
    --help this help.
    --mode={incremental,full,separate}
    --stage={build,check,doc}
    --platform={ubuntu,fedora,guile2}
""")


def parse():
    opts, args = getopt.getopt(sys.argv[1:],
                               "",
                               ["help",
                                "mode=", "platform=", "stage="])

    result = Options()
    all_platforms = set(["fedora-guile2", "fedora", "ubuntu"])
    all_stages = set(["doc", "build", "check"])
    all_modes = set(["full", "incremental", "separate"])
    for (k, v) in opts:
        if False:
            pass
        elif k == "--mode":
            if v == "incr": v = "incremental"
            if v not in all_modes:
                error ("unknown mode %s, want %s" % (v, all_modes))

            result.mode = v
        elif k == "--platform":
            if v == "guile2":
                v = "fedora-guile2"
            if v == "all":
                result.platforms.update(all_platforms)
            elif v not in all_platforms:
                error ("unknown platform %s, want %s" %(v, all_platforms))
                result.platforms.add(v)
        elif k == "--stage":
            if v not in all_stages:
                error ("unknown stage %s, want %s" % (v, all_stages))
            result.stage = v
        elif k == "--help":
            usage()
            sys.exit(0)

    return (result, args)

def test_one(options, platform, args):
    driver_script = "test-%s.sh" % options.mode
    mode = options.mode
    stage = options.stage
    if mode == "incremental":
        seed_image = "lilypond-seed-%s" % platform
    else:
        seed_image = "lilypond-base-%s" % platform


    if len(args) != 2:
        error ("need args URL BRANCH or 'rietveld' CHANGE-NUM")

    local_repo = "local"
    (url, branch) = args

    print("\n\nTesting %(url)s %(branch)s for platform %(platform)s, mode %(mode)s, stage %(stage)s" %  locals())

    container_url = "/local"
    if os.path.exists(url):
        url = os.path.abspath(url)
        system("""set -eu
        cd lilypond
        git checkout -f $(git rev-parse origin/master)
        git fetch -f %(url)s %(branch)s:%(branch)s
        """ % locals())

    if url == "rietveld":
        change = branch
        patchset = url
        js = urllib.request.urlopen("https://codereview.appspot.com/api/%s/" % change).read()
        data = json.loads(js)
        patchset = data["patchsets"][-1]
        issue  = "issue%s_%d" % (change, patchset)

        system('''cd lilypond
        git fetch origin
        git checkout -f origin/master
        (git branch -D %(issue)s || true)
        git checkout -b %(issue)s origin/master
        curl https://codereview.appspot.com/download/%(issue)s.diff | git apply
        git commit -m "%(issue)s" -a
        ''' % locals())

        version = os.popen("cd lilypond ; git rev-parse --short=8 HEAD")
        local_repo = "rietveld"
        container_url = "/rietveld"
        branch = issue

    name = re.sub("[/: ]", "-", re.sub(".*:", "", url + "_" + branch))
    cwd = os.getcwd()
    dest = os.path.join(cwd, "../lilypond-test-results", name, seed_image)
    os.makedirs(dest, exist_ok=True)
    cmd=("set -eu ; docker run -v %(dest)s:/output -v %(cwd)s/lilypond:/%(local_repo)s:ro "
           + "-v %(cwd)s/%(driver_script)s:/test.sh:ro "
           + "--rm=true "
           + "%(seed_image)s /test.sh %(stage)s %(container_url)s %(branch)s /%(local_repo)s origin/master") % locals()
    system (cmd)

    print( "results in %s" % dest)

def main():
    (opts, args) = parse()
    if not opts.platforms:
        opts.platforms.add("fedora-guile2")

    if not opts.stage:
        opts.stage = "check"
    if not opts.mode:
        opts.mode = "incremental"

    for p in opts.platforms:
        test_one(opts, p, args)

if __name__ == "__main__":
    main()
