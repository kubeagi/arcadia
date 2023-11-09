# Copyright 2023 KubeAGI.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import datetime


def now_str():
    return f"{datetime.datetime.now():%Y-%m-%d %H:%M:%S.%f}"

def now_str_for_day():
    return f"{datetime.datetime.now():%Y-%m-%d}"


def now_str_for_file_name():
    return f"{datetime.datetime.now():%Y_%m_%d_%H_%M_%S_%f}"


def timestamp_to_str(timestamp):
    return f"{datetime.datetime.fromtimestamp(timestamp):%Y-%m-%d %H:%M:%S.%f}"


def timestamp_to_str_second(timestamp):
    return f"{datetime.datetime.fromtimestamp(timestamp):%Y-%m-%d %H:%M:%S}"


def chage_datetime_fromat(opt={}):
    my_date_time = datetime.datetime.strptime(
                        opt['date_time'],
                        opt['from_format'])

    return my_date_time.strftime(opt.get('to_format', '%Y-%m-%d %H:%M:%S'))