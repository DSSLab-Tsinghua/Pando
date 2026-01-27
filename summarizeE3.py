import sys, os, time
import json, math

TOTAL_INSTANCE = 31
date=''

if  __name__ =='__main__':
   
    TotalArr = dict()
    TransmissionArr = dict()
    ConsensusArr = dict()

    f = math.floor((TOTAL_INSTANCE-1)/3)
    quorum = 2*f+1

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
                # print(item)
                if item[2] == "[Transmission]":
                    if int(item[3]) not in TransmissionArr:
                        TransmissionArr[int(item[3])] = []
                    historyTrans = TransmissionArr[int(item[3])]
                    historyTrans.append([int(item[5]),int(item[6])])
                    TransmissionArr[int(item[3])] = historyTrans
                if item[2] == "[Consensus]":
                    if int(item[3]) not in ConsensusArr:
                        ConsensusArr[int(item[3])] = []
                    historyCons = ConsensusArr[int(item[3])]
                    historyCons.append([int(item[5]),int(item[6])])
                    ConsensusArr[int(item[3])] = historyCons                        
                if item[2] == "[Total]":
                    if int(item[3]) not in TotalArr:
                        TotalArr[int(item[3])] = []
                    historyTotal = TotalArr[int(item[3])]
                    historyTotal.append([int(item[5]),int(item[6])])
                    TotalArr[int(item[3])] = historyTotal                                                
            file.close()

    #print("%s\n\n%s\n\n%s"%(TransmissionArr, ConsensusArr, TotalArr))     #{epoch:[latency, tps], ..., ...}
    #OutputDict = dict()
    TransLatency = []
    for key in sorted(TransmissionArr.keys()):
        for [latency, tps] in TransmissionArr[key]:
            TransLatency.append(latency)            
    TransLatency.sort(reverse=False)
    TransLatency = TransLatency[int(len(TransLatency)*4/5):]
    Transavglatency = sum(TransLatency)/len(TransLatency)
    print("TransLatency",TransLatency)

    ConsLatency = []
    for key in sorted(ConsensusArr.keys()):
        for [latency, tps] in ConsensusArr[key]:
            ConsLatency.append(latency)            
    ConsLatency.sort(reverse=False)
    ConsLatency = ConsLatency[int(len(ConsLatency)*4/5):]
    Consavglatency = sum(ConsLatency)/len(ConsLatency)
    print("ConsLatency",ConsLatency)

    TotalLatency = []    
    for key in sorted(TotalArr.keys()):
        for [latency, tps] in TotalArr[key]:
            TotalLatency.append(latency)
    TotalLatency.sort(reverse=False)
    TotalLatency = TotalLatency[int(len(TotalLatency)*4/5):]
    Totalavglatency = sum(TotalLatency)/len(TotalLatency)
    print("TotalLatency",TotalLatency)

    with open("resultE3.txt", 'a') as dfile:
        p = time.strftime("%Y-%m-%d-%H%m%s", time.localtime()) + " (TransLatency," + str(round(Transavglatency,2)) + " ms)" + "(ConsLatency," + str(round(Consavglatency,2)) + " ms)" + "(TotalLatency," + str(round(Totalavglatency,2)) + " ms)\n" 
        print(p)
        dfile.write(p)
    dfile.close()