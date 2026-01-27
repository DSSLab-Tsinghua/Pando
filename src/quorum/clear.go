/*
Clear quorum info
*/

package quorum

func ClearPCer(seq int){
	PCer.Delete(seq) 
}

func ClearQC(seq int){
	QC.Delete(seq) 
}