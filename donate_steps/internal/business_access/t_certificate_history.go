package business_access

import "git.code.oa.com/gongyi/donate_steps/internal/data_access"

func GetCertificateHistory(week, oid string) (data_access.CertificateHistory, error) {
	ch, err := data_access.GetCertificateHistory(week, oid)
	if err != nil {
		return ch, err
	}
	return ch, nil
}

func InsertCertificateHistory(db_proxy data_access.DBProxy, ch data_access.CertificateHistory) error {
	if err := data_access.InsertCertificateHistory(db_proxy, ch); err != nil {
		return err
	}
	return nil
}

func InsertListCertificateHistory(db_proxy data_access.DBProxy, chs []data_access.CertificateHistory) error {
	if err := data_access.InsertListCertificateHistory(db_proxy, chs); err != nil {
		return err
	}
	return nil
}
