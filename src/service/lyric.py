# -*- coding: utf-8 -*-
import requests
import urllib.parse
from bs4 import BeautifulSoup


def searchLyric(name):
    site = "https://www.taiwanbible.com"
    
    # search id
    path = "/web/search.jsp"
    url = site + path
    headers = {
        'Content-Type': 'application/x-www-form-urlencoded',
        'Cookie': 'JSESSIONID=35454E79B5902585014596FC3BC4C116'
    }
    payload='area=lyrics&keyword=' + urllib.parse.quote(name)
    r = requests.request("POST", url,headers=headers, data=payload)
    soup = BeautifulSoup(r.text,"lxml")
    searchResultTable = soup.find_all("table",class_="bordered")[1].find_all('tr')

    if len(searchResultTable) < 2:
        return ""
    else:
        path = searchResultTable[1].find_all('td')[1].find("a")['href']  # skip header and get first result
    

    # get lyric
    url = site + path
    r = requests.request("GET",url,headers=headers)
    soup = BeautifulSoup(r.text,"html.parser")

    print(soup.find_all("span",style="word-break:break-all;"))
    lyric = name+"'s lyric showed below."
    return lyric

if __name__ == '__main__':
    print(searchLyric("我要歌頌你的力量"))
    
