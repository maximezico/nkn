package transaction


import (
	"GoOnchain/common"
	"io"
	"GoOnchain/common/serialization"
)

type UTXOTxInput struct {

	//Indicate the previous Tx which include the UTXO output for usage
	ReferTxID common.Uint256

	//The index of output in the referTx output list
	ReferTxOutputIndex uint16
}


func (ui *UTXOTxInput) Serialize(w io.Writer)  {
	ui.ReferTxID.Serialize(w)
	serialization.WriteVarInt(w,ui.ReferTxOutputIndex)
}

func (ui *UTXOTxInput) Deserialize(r io.Reader) error {
	//referTxID
	err := ui.ReferTxID.Deserialize(r)
	if err != nil {return err}

	//Output Index
	err = ui.ReferTxID.Deserialize(r)
	if err != nil {return err}

	return nil
}
