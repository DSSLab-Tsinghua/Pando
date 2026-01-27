import sys, os, time
import json, math

TOTAL_INSTANCE = 31
date=''     #We use "time.localtime()"" to set the log filename. If you do not finish the test in a same day, please provide the date when you make the test. For example: date='20250714'

if  __name__ =='__main__':
   
    TotalArr = dict()
    
    f = math.floor((TOTAL_INSTANCE-1)/3)
    quorum = 2*f+1

    config = "./etc/conf.json"
    with open(config, 'r') as config_file:
        config_data = json.load(config_file)

    for ID in range(0, TOTAL_INSTANCE):
        if date == '':
            filename = "./var/log/" + str(ID) + "/" + time.strftime("%Y%m%d", time.localtime()) + "_Eva.log"
        else:
            filename = "./var/log/" + str(ID) + "/" + date + "_Eva.log"
        
        with open(filename, 'r') as file:
            file.readline()
            file.readline()
            while True:
                line = file.readline()
                if not line:
                    break
                item = line.strip().split(' ')
                if item[2] == "[Total]":
                    if int(item[3]) not in TotalArr:
                        TotalArr[int(item[3])] = []
                    historyTotal = TotalArr[int(item[3])]
                    historyTotal.append([int(item[5]),int(item[6])])
                    TotalArr[int(item[3])] = historyTotal                                                
            file.close()

    TotalLatency = []    
    for key in sorted(TotalArr.keys()):
        for [latency, tps] in TotalArr[key]:
            TotalLatency.append(latency)
    TotalLatency.sort(reverse=False)
    TotalLatency = TotalLatency[int(len(TotalLatency)*4/5):]
    Totalavglatency = sum(TotalLatency)/len(TotalLatency)
    print("TotalLatency",TotalLatency)

    TotalTPS = quorum*int(config_data['batchSize'])*1000/Totalavglatency
    with open("./resultE2.txt", 'a') as dfile:
        p = time.strftime("%Y-%m-%d-%H%m%s", time.localtime()) + " batchsize" + str(config_data['batchSize']) + ": (TotalLatency," + str(round(Totalavglatency,2)) + " ms)" + "(TotalTPS," + str(round(TotalTPS,2)) + " tx/s)\n"
        print(p)
        dfile.write(p)
    dfile.close()
