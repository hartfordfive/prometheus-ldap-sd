#!/usr/bin/env python3

import requests
import yaml
import sys
import time
import logging
from optparse import OptionParser


logging.basicConfig(
  format='%(asctime)s %(levelname)-8s %(message)s',
  level=logging.INFO,
  datefmt='%Y-%m-%d %H:%M:%S')


parser = OptionParser()
 
parser.add_option("-H", "--host",
                  dest = "host",
                  default = "127.0.0.1",
                  help = "Server host address")
parser.add_option("-p", "--port",
                  dest = "port",
                  default = "80",
                  help = "Server port")
parser.add_option("-c", "--config",
                  dest = "config",
                  default = "./prometheus-ldap-sd.yaml",
                  help = "Scrape interval")
parser.add_option("-i", "--scrape_interval",
                  dest = "scrape_interval",
                  default = 60,
                  help = "Scrape interval")
parser.add_option("-w", "--wait",
                  dest = "wait",
                  default = 0.2,
                  help = "Period to wait between target group scrapes")
 
(options, args) = parser.parse_args()


conf = {}
with open(options.config, "r") as stream:
    try:
        conf = (yaml.safe_load(stream))
    except yaml.YAMLError as e:
        print(e)


while True:
  print(f"--------------------- Scraping {len(conf['ldap_config']['base_dn_mappings'])} target groups -------------------")
  for k,v in conf['ldap_config']['base_dn_mappings'].items():
    url = f"http://{options.host}:{options.port}/targets?targetGroup={k}"
    logging.info(f"Fetching targets for group {k}")
    res = requests.get(url)
    logging.info(f"\tURL: {url} (Total size: {len(res.content)})")
    time.sleep(float(options.wait))
  print("\n")
  time.sleep(float(options.scrape_interval))