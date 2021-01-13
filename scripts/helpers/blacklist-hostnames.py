import json
import sys
from typing import List

import requests

# This script is used to import list of hostnames to 'blacklist'
# it will pull hostnames from the CT log source (see url variable) & custom define ones
# and blacklist them to prevent useless crawling

url = "https://raw.githubusercontent.com/alecmuffett/real-world-onion-sites/master/ct-log.txt"
custom_hostnames = [
    'gamebombfak3pwnh.onion',  # gaming forum, lot of noise
    'metagerv65pwclop2rsfzg4jwowpavpwd6grhhlvdgsswvo6ii4akgyd.onion'  # search engine, lot of noise
]
config_api_uri = sys.argv[1]


def add_if_not_exist(a: List[dict], b: str):
    found = False
    for i in a:
        if i['hostname'] == b:
            found = True

    if not found:
        a.append({'hostname': b})


# Get up-to-date list of real-world / legit .onion
r = requests.get(url)
new_hostnames = []
for hostname in r.text.splitlines():
    new_hostnames.append({'hostname': hostname})
print("pulled {} real world hostnames from ct-log.txt".format(len(new_hostnames)))

# Append custom hostnames ignore list
for custom_hostname in custom_hostnames:
    add_if_not_exist(new_hostnames, custom_hostname)
print("added {} custom hostnames".format(len(custom_hostnames)))

# Query existing blacklisted hostnames from ConfigAPI
r = requests.get(config_api_uri + "/config/forbidden-hostnames")
forbidden_hostnames = r.json()
print("there is {} forbidden hostnames defined in ConfigAPI".format(len(forbidden_hostnames)))

# Merge the lists while preventing duplicates
for forbidden_hostname in forbidden_hostnames:
    add_if_not_exist(new_hostnames, forbidden_hostname['hostname'])
print("there is {} forbidden hostnames now".format(len(new_hostnames)))

# Update ConfigAPI
headers = {'Content-Type': 'application/json', 'Accept': 'application/json'}
r = requests.put(config_api_uri + "/config/forbidden-hostnames", json.dumps(new_hostnames), headers=headers)

if r.ok:
    print("successfully updated forbidden hostnames")
