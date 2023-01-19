# -*- coding: utf-8 -*-
from concurrent import futures
import logging

import grpc
import service_pb2
import service_pb2_grpc


def searchLyric(name):
    lyric = name+"'s lyric showed below."
    return lyric


class service(service_pb2_grpc.MyServiceServicer):
    def SearchLyric(self, request, context):
        lyric = searchLyric(request.name)
        return service_pb2.songinfo(lyric=lyric)


def serve():
    port = '50051'
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    service_pb2_grpc.add_MyServiceServicer_to_server(service(), server)
    server.add_insecure_port('[::]:' + port)
    server.start()
    print("Server started, listening on " + port)
    server.wait_for_termination()


if __name__ == '__main__':
    logging.basicConfig()
    serve()
    #print(searchLyric("song name"))