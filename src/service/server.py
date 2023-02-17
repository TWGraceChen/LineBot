# -*- coding: utf-8 -*-
from concurrent import futures
import logging
import sys
import grpc
import service_pb2
import service_pb2_grpc
import lyric
import ppt



class service(service_pb2_grpc.MyServiceServicer):
    def SearchLyric(self, request, context):
        result = lyric.searchLyric(request.name)
        return service_pb2.songinfo(lyric=result)
    def MakePpt(self, request, context):
        filename = ppt.makeppt(request.songnames,request.lyrics)
        return service_pb2.filename(filename=filename)

def serve(port):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    service_pb2_grpc.add_MyServiceServicer_to_server(service(), server)
    server.add_insecure_port('[::]:' + port)
    server.start()
    print("gRPC Server started, listening on " + port)
    server.wait_for_termination()


if __name__ == '__main__':
    if len(sys.argv) < 2:
        port = '50051'
    else:
        port = sys.argv[1]
    
    logging.basicConfig()
    serve(port)