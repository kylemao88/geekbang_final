//
package match

// RecoverYQZMatchState use for new redis or recover old activity
func RecoverYQZMatchState(matchID string) error {
	/*
		atomic.AddInt32(&SpeedRecoverMatchRankTime, 1)
		// step1 get all activity match record in user status
		matchInfo, err := fetchMatchStateFromDB(matchID)
		if err != nil {
			logger.Error("fetchMatchStateFromDB error: %v", err)
			return err
		}
		// we need add useless user in it avoid read db again
		err = updateMatchState(matchID)
		if err != nil {
			logger.Error("add default item to activity: %v rank error: %v", matchID, err)
			return err
		}
		logger.Info("recover activity: %v record count: %v success", matchID, len(ranks))
	*/
	return nil
}

