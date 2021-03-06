# Copyright (c) 2020, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
#
# WSO2 Inc. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.

import atexit
import ipaddress
import os
import pickle
import random
import string
import sys
import time
from datetime import datetime
from multiprocessing import Process

import numpy as np
import requests
import yaml

from constants import *
from utils import util_methods
from utils import log


# noinspection PyProtectedMember
def generate_unique_ip():
    """
    Returns a unique ip address
    :return: an unique ip
    """
    global used_ips

    random.seed()
    MAX_IPV4 = ipaddress.IPv4Address._ALL_ONES
    temp_ip = ipaddress.IPv4Address._string_from_ip_int(random.randint(0, MAX_IPV4))
    while temp_ip in used_ips:
        temp_ip = ipaddress.IPv4Address._string_from_ip_int(random.randint(0, MAX_IPV4))

    used_ips.append(temp_ip)
    return temp_ip


def generate_cookie():
    """
    generates a random cookie
    :return: a randomly generated cookie
    """
    letters_and_digits = string.ascii_lowercase + string.digits
    cookie = 'JSESSIONID='
    cookie += ''.join(random.choice(letters_and_digits) for _ in range(31))
    return cookie


def simulate_user(user_data):
    """
      Simulate the behaviour of a user during the attack duration.
      :param user_data: A dictionary containing the user data
      :return: None
      """
    global attack_duration, protocol, host, port, payloads, user_agents, start_time, dataset_path, invoke_patterns

    up_time = datetime.now() - start_time
    sleep_pattern = invoke_patterns[random.choice(list(invoke_patterns.keys()))]

    if up_time.seconds < attack_duration:
        for app in user_data.values():

            invoke_pattern_indices = util_methods.generate_method_invoke_pattern(app)

            for i in invoke_pattern_indices:
                up_time = datetime.now() - start_time

                if up_time.seconds >= attack_duration:
                    break

                sleep_time = np.absolute(np.random.normal(sleep_pattern['mean'], sleep_pattern['std']))
                time.sleep(sleep_time)

                scenario = app[i]
                invoke_path = scenario[2]
                token = scenario[3]
                http_method = scenario[4]
                request_path = "{}://{}:{}/{}".format(protocol, host, port, invoke_path)
                random_user_agent = random.choice(user_agents)
                random_ip = generate_unique_ip()
                random_cookie = generate_cookie()
                random_payload = random.choice(payloads)
                accept = content_type = "application/json"

                try:
                    response = util_methods.send_simple_request(request_path, http_method, token, random_ip, random_cookie, accept, content_type, random_user_agent, payload=random_payload)
                    request_info = "{},{},{},{},{},{},{},{},{},\"{}\",{}".format(datetime.now(), random_ip, token, http_method, request_path, random_cookie, accept, content_type, random_ip,
                                                                                 random_user_agent,
                                                                                 response.status_code,
                                                                                 )
                    util_methods.write_to_file(dataset_path, request_info, "a")
                except requests.exceptions.ConnectionError as e:
                    error_code = 521
                    request_info = "{},{},{},{},{},{},{},{},{},\"{}\",{}".format(datetime.now(), random_ip, token, http_method, request_path, random_cookie, accept, content_type, random_ip,
                                                                                 random_user_agent,
                                                                                 error_code,
                                                                                 )
                    util_methods.write_to_file(dataset_path, request_info, "a")
                    logger.error("Connection Error: {}".format(e))
                except requests.exceptions.RequestException:
                    logger.exception("Request Failure")


# Program Execution
if __name__ == '__main__':

    logger = log.set_logger("Stolen_TOKEN")

    # Constants
    STOLEN_TOKEN = 'stolen_token'

    try:
        with open(os.path.abspath(os.path.join(__file__, "../../../../traffic-tool/data/runtime_data/scenario_pool.sav")), "rb") as scenario_file:
            scenario_pool = pickle.load(scenario_file, )

        with open(os.path.abspath(os.path.join(__file__, "../../../../../config/attack-tool.yaml")), "r") as attack_config_file:
            attack_config = yaml.load(attack_config_file, Loader=yaml.FullLoader)

    except FileNotFoundError as ex:
        logger.error("{}: \'{}\'".format(ex.strerror, ex.filename))
        sys.exit()

    # Reading configurations from attack-tool.yaml
    protocol = attack_config[GENERAL_CONFIG][API_HOST][PROTOCOL]
    host = attack_config[GENERAL_CONFIG][API_HOST][IP]
    port = attack_config[GENERAL_CONFIG][API_HOST][PORT]
    attack_duration = attack_config[GENERAL_CONFIG][ATTACK_DURATION]
    payloads = attack_config[GENERAL_CONFIG][PAYLOADS]
    user_agents = attack_config[GENERAL_CONFIG][USER_AGENTS]
    process_count = attack_config[GENERAL_CONFIG][NUMBER_OF_PROCESSES]
    compromised_user_count = attack_config[ATTACKS][STOLEN_TOKEN][COMPROMISED_USER_COUNT]
    invoke_patterns = util_methods.process_time_patterns(attack_config[GENERAL_CONFIG][TIME_PATTERNS])

    # Recording column names in the dataset csv file
    dataset_path = "../../../../../../dataset/attack/stolen_token.csv"
    util_methods.write_to_file(dataset_path, "timestamp,ip_address,access_token,http_method,invoke_path,cookie,accept,content_type,x_forwarded_for,user_agent,response_code", "w")

    used_ips = []
    start_time = datetime.now()

    logger.info("Stolen token attack started")

    if compromised_user_count > len(scenario_pool):
        logger.error("More compromised users than the total users")
        sys.exit()

    compromised_users = np.random.choice(list(scenario_pool.values()), size=compromised_user_count, replace=False)
    process_list = []

    for user in compromised_users:
        process = Process(target=simulate_user, args=(user,))
        process.daemon = False
        process_list.append(process)
        process.start()

        with open(os.path.abspath(os.path.join(__file__, "../../../data/runtime_data/attack_processes.pid")), "a+") as f:
            f.write(str(process.pid) + '\n')

    while True:
        time_elapsed = datetime.now() - start_time
        if time_elapsed.seconds >= attack_duration:
            for process in process_list:
                process.terminate()
            
            with open(os.path.abspath(os.path.join(__file__, "../../../data/runtime_data/attack_processes.pid")), "w") as f:
                f.write('')

            logger.info("Attack terminated successfully. Time elapsed: {} minutes".format(time_elapsed.seconds / 60.0))
            break

    # cleaning up the processes at exit
    atexit.register(util_methods.cleanup, process_list=process_list)
