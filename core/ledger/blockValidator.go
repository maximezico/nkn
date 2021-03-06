package ledger

import (
	"errors"
	"fmt"

	tx "github.com/nknorg/nkn/core/transaction"
	"github.com/nknorg/nkn/core/validation"
	. "github.com/nknorg/nkn/errors"
)

func VerifyBlock(block *Block, ld *Ledger, completely bool) error {
	if block.Header.Height == 0 {
		return nil
	}
	err := VerifyBlockData(block.Header, ld)
	if err != nil {
		return err
	}

	flag, err := validation.VerifySignableData(block)
	if flag == false || err != nil {
		return err
	}

	if block.Transactions == nil {
		return errors.New(fmt.Sprintf("No Transactions Exist in Block."))
	}
	if block.Transactions[0].TxType != tx.Coinbase {
		return errors.New(fmt.Sprintf("Header Verify failed first Transacion in block is not Coinbase type."))
	}
	for index, v := range block.Transactions {
		if v.TxType == tx.Coinbase && index != 0 {
			return errors.New(fmt.Sprintf("This Block Has Coinbase transaction after first transaction in block."))
		}
	}

	//verfiy block's transactions
	if completely {
		/*
			//TODO: NextBookKeeper Check.
			bookKeeperaddress, err := ledger.GetBookKeeperAddress(ld.Blockchain.GetBookKeepersByTXs(block.Transactions))
			if err != nil {
				return errors.New(fmt.Sprintf("GetBookKeeperAddress Failed."))
			}
			if block.Header.NextBookKeeper != bookKeeperaddress {
				return errors.New(fmt.Sprintf("BookKeeper is not validate."))
			}
		*/
		for _, txVerify := range block.Transactions {
			if errCode := tx.VerifyTransaction(txVerify); errCode != ErrNoError {
				return errors.New(fmt.Sprintf("VerifyTransaction failed when verifiy block"))
			}
			if errCode := tx.VerifyTransactionWithLedger(txVerify); errCode != ErrNoError {
				return errors.New(fmt.Sprintf("VerifyTransactionWithLedger failed when verifiy block"))
			}
		}
		if err := tx.VerifyTransactionWithBlock(block.Transactions); err != nil {
			return errors.New(fmt.Sprintf("VerifyTransactionWithBlock failed when verifiy block"))
		}
	}

	return nil
}

func VerifyHeader(bd *Header, ledger *Ledger) error {
	return VerifyBlockData(bd, ledger)
}

func VerifyBlockData(bd *Header, ledger *Ledger) error {
	if bd.Height == 0 {
		return nil
	}

	prevHeader, err := ledger.Blockchain.GetHeader(bd.PrevBlockHash)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "[BlockValidator], Cannnot find prevHeader..")
	}
	if prevHeader == nil {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], Cannnot find previous block.")
	}

	if prevHeader.Height+1 != bd.Height {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block height is incorrect.")
	}

	if prevHeader.Timestamp >= bd.Timestamp {
		return NewDetailErr(errors.New("[BlockValidator] error"), ErrNoCode, "[BlockValidator], block timestamp is incorrect.")
	}

	return nil
}
