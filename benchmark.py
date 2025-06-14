#!/usr/bin/python


"""
Script for structured benchmarking of lilypond binaries. To use, compile like this:


$ make -C lily && test -z "$(git diff )" && cp lily/out/lilypond binaries/lilypond.$(git rev-parse  --short HEAD) && ln binaries/* out/bin/

then

$ python scripts/auxiliar/benchmark.py -n3  -v HEAD^ -v HEAD -- input/regression/mozart-hrn-3.ly


The first -v should be the baseline, the second should be what you want to test.
"""

import math
import re
import os
import sys
import statistics
import getopt

def error(msg):
  sys.stdout.write("%s\n" % msg)
  sys.exit(1)

def system(cmd, allow_error=False):
  print("running %s" % cmd)
  status = os.system(cmd)
  if status and not allow_error:
    error("failure running %s" % cmd)
  return status

class Options:
  pass


def parse_cmdline():
  options = Options()
  options.run_count = 3
  options.versions = []
  options.descriptions = {}
  options.out_dir = "benchmark-results"

  opts, args = getopt.getopt(sys.argv[1:], "n:b:v:",[])

  baseline = ''
  testversion = ''
  for (o, a) in opts:
    if o == '-n':
      options.run_count = int(a)
    if o == '-b':
      baseline = a
    if o == '-v':
      commit = os.popen("git rev-parse --short %s" % a).read().strip()
      testversion = commit

  if not baseline:
    baseline = os.popen("git rev-parse --short %s^" % testversion).read().strip()

  options.versions = [baseline, testversion]
  for v in options.versions:
      descr = os.popen("git log -1 --pretty=format:%%s %s" % v).read()
      options.descriptions[v] = descr

  options.args = args
  return options

def sanity_check():
  if '-O2' not in open('config.make').read():
    error("need -O2 in configuration")

  diff = os.popen("git diff").read()
  if diff:
    error("must have clean tree")

def get_branch():
  cur_branch = ""
  full_ref = os.popen("git rev-parse --symbolic-full-name HEAD").read()
  if full_ref.startswith("refs/heads/"):
    cur_branch = full_ref[len("refs/heads/"):]
  return cur_branch

def build_versions(versions):
  for v in versions:
    if not os.path.exists("binaries/lilypond.%s" % v):
      system("git checkout %s" % v)
      system("make -C flower/ clean && make -C lily/ clean && make -C flower && make -C lily/ -j4")
      system("cp lily/out/lilypond binaries/lilypond.%s" % v)
    if not os.path.exists("out/bin/lilypond.%s" % v):
      system("ln binaries/lilypond.%s out/bin/lilypond.%s" % (v, v))


def command_id (options):
  return  re.sub('[^a-zA-Z0-9_-]', '', '-'.join(options.args))

def run_timings(options):
  os.makedirs(options.out_dir, exist_ok=True)

  memory_results = {}
  timing_results = {}
  for v in options.versions:
    timing_results[v] = []
    memory_results[v] = []


  for run in range(0, options.run_count):
    for v in options.versions:
      out = '%s/%s-%s.%d.txt' % (options.out_dir, command_id(options), v, run)
      system("git checkout %s" % v)
      status  = system("/usr/bin/time -v out/bin/lilypond.%s %s >& %s" % (v, ' '.join(options.args), out))

      if status:
        system("git checkout %s" % cur_branch)
        error("lilypond run failed")

      for l in open(out).readlines():
        m = re.search('User time \(seconds\): ([0-9.]+)', l)
        if m:
          timing_results[v].append(float(m.group(1)))

        m = re.search('Maximum resident set size.*: ([0-9.]+)', l)
        if m:
          memory_results[v].append(float(m.group(1)))
  return memory_results, timing_results

def stddev(l):
  if len(l) > 1:
    return statistics.stdev(l)
  return float("+inf")

def abs_summary(options, memory_results, timing_results):
  result = ""

  result += "benchmark for arguments: %s\n" % ' '.join(options.args)

  result += "raw data:\n  mem %s\n  time %s\n" % (memory_results, timing_results)

  for v in options.versions:
    timings = timing_results[v]
    memory = memory_results[v]
    result +=("Version %s: %s\n" % (v, options.descriptions[v]))
    result += ("  time avg %f\n" % statistics.mean(timings))
    result += ("  time med %f\n" % statistics.median(timings))
    result += ("  time stddev %.3f\n" % stddev(timings))

    result += ("  mem  avg %f\n" % statistics.mean(memory))
    result += ("  mem  med %f\n" % statistics.median(memory))
    result += ("  mem  stddev %.3f\n" % stddev(memory))
  return result

def delta_summary_metric(base_version, version, run_count, metric_name, metric):
  base = metric[base_version]
  data = metric[version]
  result = ""
  delta =  statistics.median(data) - statistics.median(base)
  if metric_name == "memory":
    prec = 0
    qual = ((delta > 0) and "fatter") or "leaner"
  elif metric_name == "time":
    prec = 2
    qual = ((delta > 0) and "slower") or "faster"
  else:
    assert 0, metric_name

  if math.fabs(delta) < (stddev(base) + stddev(base)):
    qual = "neutral"

  result += "  %s: med diff %.*f (stddevs %.*f %.*f, n=%d)\n" % (metric_name, prec, delta, prec, stddev(base), prec, stddev(data), run_count)
  result += "  %s: med diff %.1f %% (%s is %s)\n" % (metric_name, delta / statistics.median(base) * 100.0, version, qual)
  return result

def delta_summary_version(options, v, memory_results, time_results):
  result = ""
  result += "%s - %s\n" %(v, options.descriptions[v])
  result += "  baseline: %s %s\n" %(options.versions[0], options.descriptions[options.versions[0]])
  result += "  args: %s\n" %(' '.join(options.args))
  result += delta_summary_metric(options.versions[0], v, options.run_count, "memory", memory_results)
  result += delta_summary_metric(options.versions[0], v, options.run_count, "time", time_results)
  return result



def main():
  options = parse_cmdline()
  print(options.descriptions)
  sanity_check()

  build_versions(options.versions)

  mem, time = run_timings(options)

  result = abs_summary(options, mem, time)
  for v in options.versions[1:]:
    result += delta_summary_version(options, v, mem, time)

  open("%s/%s-%s-summary.txt" % (options.out_dir, command_id(options), '-'.join(options.versions[1:])), 'w').write(result)

  sys.stdout.write(result)
  print()

cur_branch = get_branch()
try:
  main()
finally:
  system("git checkout %s" % cur_branch)
