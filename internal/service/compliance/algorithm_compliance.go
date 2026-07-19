package compliance

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (a *App) handleCreateContentReport(c *gin.Context) {
	var req struct {
		TargetType         string `json:"target_type"`
		TargetID           uint   `json:"target_id"`
		GenerationRecordID uint   `json:"generation_record_id"`
		WorkID             uint   `json:"work_id"`
		Reason             string `json:"reason"`
		Description        string `json:"description"`
		Contact            string `json:"contact"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "è¯·æ±‚æ ¼å¼é”™è¯¯")
		return
	}
	targetType := strings.TrimSpace(req.TargetType)
	reason := strings.TrimSpace(req.Reason)
	if targetType == "" || req.TargetID == 0 || reason == "" {
		writeError(c, http.StatusBadRequest, "invalid_content_report", "ä¸¾æŠ¥å¯¹è±¡å’ŒåŽŸå› ä¸èƒ½ä¸ºç©º")
		return
	}

	user := currentUser(c)
	report := ContentReport{
		UserID:      &user.ID,
		TargetType:  targetType,
		TargetID:    req.TargetID,
		Reason:      reason,
		Description: strings.TrimSpace(req.Description),
		Contact:     strings.TrimSpace(req.Contact),
		Status:      ContentReportStatusPending,
	}
	if req.GenerationRecordID > 0 {
		report.GenerationRecordID = &req.GenerationRecordID
	}
	if req.WorkID > 0 {
		report.WorkID = &req.WorkID
	}

	review := ContentSafetyReview{
		ReviewType:         ContentReviewTypeShare,
		Status:             ContentSafetyStatusPending,
		RiskLevel:          "medium",
		Reason:             reason,
		TargetType:         targetType,
		TargetID:           req.TargetID,
		GenerationRecordID: report.GenerationRecordID,
		WorkID:             report.WorkID,
		UserID:             &user.ID,
		InputSummary:       strings.TrimSpace(req.Description),
	}
	if report.GenerationRecordID != nil {
		var generation GenerationRecord
		if err := a.db.First(&generation, *report.GenerationRecordID).Error; err == nil {
			review.Model = fallbackString(generation.RuntimeModel, generation.Model)
			review.ProviderRequestID = generation.ProviderRequestID
			if review.InputSummary == "" {
				review.InputSummary = generation.Prompt
			}
		}
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&review).Error; err != nil {
			return err
		}
		report.ContentReviewID = &review.ID
		return tx.Create(&report).Error
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "content_report_create_failed", "ä¸¾æŠ¥æäº¤å¤±è´¥")
		return
	}

	writeJSON(c, http.StatusCreated, gin.H{"id": report.ID, "status": report.Status})
}

func (a *App) handleListAdminContentReviews(c *gin.Context) {
	page, pageSize := compliancePage(c)
	query := a.db.Model(&ContentSafetyReview{})
	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if reviewType := strings.TrimSpace(c.Query("review_type")); reviewType != "" && reviewType != "all" {
		query = query.Where("review_type = ?", reviewType)
	}
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + escapeLike(keyword) + "%"
		query = query.Where("reason LIKE ? OR input_summary LIKE ? OR provider_request_id LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "content_reviews_load_failed", "å†…å®¹å®¡æ ¸è¯»å–å¤±è´¥")
		return
	}
	var items []ContentSafetyReview
	if err := query.Order("created_at desc, id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&items).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "content_reviews_load_failed", "å†…å®¹å®¡æ ¸è¯»å–å¤±è´¥")
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (a *App) handlePatchAdminContentReview(c *gin.Context) {
	var review ContentSafetyReview
	if err := a.db.First(&review, c.Param("id")).Error; err != nil {
		writeError(c, http.StatusNotFound, "content_review_not_found", "å†…å®¹å®¡æ ¸ä¸å­˜åœ¨")
		return
	}
	var req struct {
		Status  string `json:"status"`
		Action  string `json:"action"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "è¯·æ±‚æ ¼å¼é”™è¯¯")
		return
	}
	status := strings.TrimSpace(req.Status)
	if !isContentSafetyStatus(status) {
		writeError(c, http.StatusBadRequest, "invalid_review_status", "å®¡æ ¸çŠ¶æ€æ— æ•ˆ")
		return
	}
	admin := currentAdmin(c)
	now := time.Now()
	review.Status = status
	review.DecisionComment = strings.TrimSpace(req.Comment)
	review.ReviewerAdminID = &admin.ID
	review.ReviewedAt = &now

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&review).Error; err != nil {
			return err
		}
		reportUpdates := map[string]any{
			"handled_by_admin_id": admin.ID,
			"handled_at":          now,
		}
		action := strings.TrimSpace(req.Action)
		if action != "" {
			reportUpdates["resolution"] = action
		}
		switch status {
		case ContentSafetyStatusReject:
			reportUpdates["status"] = ContentReportStatusResolved
		case ContentSafetyStatusPass:
			reportUpdates["status"] = ContentReportStatusRejected
		}
		if len(reportUpdates) > 2 || action != "" {
			if err := tx.Model(&ContentReport{}).Where("content_review_id = ?", review.ID).Updates(reportUpdates).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "content_review_save_failed", "å†…å®¹å®¡æ ¸ä¿å­˜å¤±è´¥")
		return
	}
	a.writeAdminAudit(c, "content_review.update", "content_safety_review", review.ID, gin.H{"status": review.Status})
	writeJSON(c, http.StatusOK, review)
}

func (a *App) handleListAdminContentReports(c *gin.Context) {
	page, pageSize := compliancePage(c)
	query := a.db.Model(&ContentReport{})
	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("q")); keyword != "" {
		like := "%" + escapeLike(keyword) + "%"
		query = query.Where("reason LIKE ? OR description LIKE ? OR resolution LIKE ? OR contact LIKE ?", like, like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "content_reports_load_failed", "ä¸¾æŠ¥è®°å½•è¯»å–å¤±è´¥")
		return
	}
	var items []ContentReport
	if err := query.Order("created_at desc, id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&items).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "content_reports_load_failed", "ä¸¾æŠ¥è®°å½•è¯»å–å¤±è´¥")
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (a *App) handleGetAdminAlgorithmDisclosure(c *gin.Context) {
	disclosure, err := a.ensureDefaultAlgorithmDisclosure()
	if err != nil {
		writeError(c, http.StatusInternalServerError, "algorithm_disclosure_load_failed", "ç®—æ³•å…¬ç¤ºè¯»å–å¤±è´¥")
		return
	}
	writeJSON(c, http.StatusOK, disclosure)
}

func (a *App) handlePatchAdminAlgorithmDisclosure(c *gin.Context) {
	disclosure, err := a.ensureDefaultAlgorithmDisclosure()
	if err != nil {
		writeError(c, http.StatusInternalServerError, "algorithm_disclosure_load_failed", "ç®—æ³•å…¬ç¤ºè¯»å–å¤±è´¥")
		return
	}
	var req AlgorithmDisclosure
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "è¯·æ±‚æ ¼å¼é”™è¯¯")
		return
	}
	if strings.TrimSpace(req.AlgorithmName) != "" {
		disclosure.AlgorithmName = strings.TrimSpace(req.AlgorithmName)
	}
	if strings.TrimSpace(req.AlgorithmType) != "" {
		disclosure.AlgorithmType = strings.TrimSpace(req.AlgorithmType)
	}
	disclosure.ServiceDescription = fallbackString(strings.TrimSpace(req.ServiceDescription), disclosure.ServiceDescription)
	disclosure.ProviderDescription = fallbackString(strings.TrimSpace(req.ProviderDescription), disclosure.ProviderDescription)
	disclosure.GovernanceSummary = fallbackString(strings.TrimSpace(req.GovernanceSummary), disclosure.GovernanceSummary)
	disclosure.MarkingSummary = fallbackString(strings.TrimSpace(req.MarkingSummary), disclosure.MarkingSummary)
	disclosure.UserRightsSummary = fallbackString(strings.TrimSpace(req.UserRightsSummary), disclosure.UserRightsSummary)
	disclosure.DisclosureVersion = fallbackString(strings.TrimSpace(req.DisclosureVersion), disclosure.DisclosureVersion)
	if strings.TrimSpace(req.Status) != "" {
		if req.Status != AlgorithmDisclosureStatusDraft && req.Status != AlgorithmDisclosureStatusPublished {
			writeError(c, http.StatusBadRequest, "invalid_disclosure_status", "ç®—æ³•å…¬ç¤ºçŠ¶æ€æ— æ•ˆ")
			return
		}
		disclosure.Status = req.Status
	}
	if err := a.db.Save(&disclosure).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "algorithm_disclosure_save_failed", "ç®—æ³•å…¬ç¤ºä¿å­˜å¤±è´¥")
		return
	}
	a.writeAdminAudit(c, "algorithm_disclosure.update", "algorithm_disclosure", disclosure.ID, gin.H{"vew7W&RäF—66Æ÷7W&UfW'6–öçÒ —w&—FT¥4ôâ†2Â‡GGå7FGW4ô²ÂF—66Æ÷7W&R§Ð ¦gVæ2†¤’†æFÆTW‡÷'DFÖ–äÆv÷&—F†Ô6ö×Æ–æ6R†2¦v–âä6öçFW‡B’° –F—66Æ÷7W&RÂW'"£ÒæVç7W&TFVfVÇDÆv÷&—F†ÔF—66Æ÷7W&R‚ ––bW'"Òæ–Â° —w&—FTW'&÷"†2Â‡GGå7FGW4–çFW&æÅ6W'fW$W'&÷"Â&Æv÷&—F†Õö6ö×Æ–æ6UöW‡÷'Eöf–ÆVB"Â.zé~k9^YŽŠxNZûÎX{®ZK‹JR" —&WGW&à —Ð —f"VæF–æu&Wf–Ww2Â&V¦V7FVE&Wf–Ww2Â&W÷'G2Â–æ6–FVçG2–çCc@ •òÒæF"äÖöFVÂ‚d6öçFVçE6fWG•&Wf–Ww·Ò’åv†W&R‚'7FGW2Òò"Â6öçFVçE6fWG•7FGW5VæF–ær’ä6÷VçB‚gVæF–æu&Wf–Ww2’äW'&÷  •òÒæF"äÖöFVÂ‚d6öçFVçE6fWG•&Wf–Ww·Ò’åv†W&R‚'7FGW2Òò"Â6öçFVçE6fWG•7FGW5&V¦V7B’ä6÷VçB‚g&V¦V7FVE&Wf–Ww2’äW'&÷  •òÒæF"äÖöFVÂ‚d6öçFVçE&W÷'G·Ò’ä6÷VçB‚g&W÷'G2’äW'&÷  •òÒæF"äÖöFVÂ‚dÆv÷&—F†Ô–æ6–FVçG·Ò’ä6÷VçB‚f–æ6–FVçG2’äW'&÷   —f"G&6W2µÔvVæW&F–öå&V6÷&@ •òÒæF"ä÷&FW"‚&7&VFVEöBFW62Â–BFW62"’äÆ–Ö—Bƒ’äf–æB‚gG&6W2’äW'&÷  —G&6T—FV×2£ÒÖ¶R…µÖv–âä‚ÂÂÆVâ‡G&6W2’ –f÷"òÂG&6R£Ò&ævRG&6W2° —G&6T—FV×2ÒVæB‡G&6T—FV×2Âv–âä‡° ’&vVæW&F–öå÷&V6÷&Eö–B#¢G&6Rä”BÀ ’'W6W%ö–B#¢G&6RåW6W$”BÀ ’&ÖöFVÂ#¢fÆÆ&6µ7G&–ær‡G&6Rå'VçF–ÖTÖöFVÂÂG&6RäÖöFVÂ’À ’'&÷f–FW%÷&WVW7Eö–B#¢G&6Rå&÷f–FW%&WVW7D”BÀ ’'7FGW2#¢G&6Rå7FGW2À ’&7&VFVEöB#¢G&6Rä7&VFVDBÀ —Ò —Ð  —w&—FT¥4ôâ†2Â‡GGå7FGW4ô²Âv–âä‡° ’&Æv÷&—F†ÕöF—66Æ÷7W&R#¢F—66Æ÷7W&RÀ ’&6öçFVçE÷&Wf–Wu÷7VÖÖ'’#¢v–âä‡° ’'VæF–ær#¢VæF–æu&Wf–Ww2À ’'&V¦V7FVB#¢&V¦V7FVE&Wf–Ww2À —ÒÀ ’&6öçFVçE÷&W÷'E÷7VÖÖ'’#¢v–âä‡° ’'F÷FÂ#¢&W÷'G2À —ÒÀ ’&–æ6–FVçE÷7VÖÖ'’#¢v–âä‡° ’'F÷FÂ#¢–æ6–FVçG2À —ÒÀ ’&vVæW&F–öå÷G&6U÷6×ÆW2#¢G&6T—FV×2À ’&Wf–FVæ6Uöæ÷FW2#¢µ×7G&–æw° ’.ZHÞyJ‚vVæW&F–öå÷&V6÷&G>8vVæW&F–öåöWfVçEöÆöw>87—7FVÕ÷&WVW7EöÆöw2Y(ÂFÖ–åöVF—EöÆöw2KÙÎK‹®ŠøhÚî[©^[ªr"À ’.Y»îx˜~yIþh‰YŽh‰iÈÞXª™ÈYÊŽš(NŠxŽ8Kˆ¾‹ÛÞY(ÎXZÎ[ÈXˆnKª¾š^[^zK¢’yIþh‰j~ŠønûÈÎ[›n˜	®‹ø~Šë[ÙR”B‹ûÞkªò"À —ÒÀ —Ò§Ð ¦gVæ2†¤’†æFÆTÆ—7DFÖ–äÆv÷&—F†Ô–æ6–FVçG2†2¦v–âä6öçFW‡B’° —vRÂvU6—¦R£Ò6ö×Æ–æ6UvR†2 —VW'’£ÒæF"äÖöFVÂ‚dÆv÷&—F†Ô–æ6–FVçG·Ò ––b7FGW2£Ò7G&–æw2åG&–Õ76R†2åVW'’‚'7FGW2"’“²7FGW2Ò""bb7FGW2Ò&ÆÂ"° —VW'’ÒVW'’åv†W&R‚'7FGW2Òò"Â7FGW2 —Ð —f"F÷FÂ–çCc@ ––bW'"£ÒVW'’ä6÷VçB‚gF÷FÂ’äW'&÷#²W'"Òæ–Â° —w&—FTW'&÷"†2Â‡GGå7FGW4–çFW&æÅ6W'fW$W'&÷"Â&Æv÷&—F†Õö–æ6–FVçG5öÆöEöf–ÆVB"Â.[©Nh
^K¨¾K»nŠû¾XùnZK‹JR" —&WGW&à —Ð —f"—FV×2µÔÆv÷&—F†Ô–æ6–FVç@ ––bW'"£ÒVW'’ä÷&FW"‚&7&VFVEöBFW62Â–BFW62"’äÆ–Ö—B‡vU6—¦R’äöfg6WB‚‡vRÒ’¢vU6—¦R’äf–æB‚f—FV×2’äW'&÷#²W'"Òæ–Â° —w&—FTW'&÷"†2Â‡GGå7FGW4–çFW&æÅ6W'fW$W'&÷"Â&Æv÷&—F†Õö–æ6–FVçG5öÆöEöf–ÆVB"Â.[©Nh
^K¨¾K»nŠû¾XùnZK‹JR" —&WGW&à —Ð —w&—FT¥4ôâ†2Â‡GGå7FGW4ô²Âv–âä‡²&—Fize": pageSize})
}

func (a *App) handleCreateAdminAlgorithmIncident(c *gin.Context) {
	var req AlgorithmIncident
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "è¯·æ±‚æ ¼å¼é”™è¯¯")
		return
	}
	incident := AlgorithmIncident{
		Title:       strings.TrimSpace(req.Title),
		Severity:    fallbackString(strings.TrimSpace(req.Severity), "medium"),
		Status:      fallbackString(strings.TrimSpace(req.Status), AlgorithmIncidentStatusOpen),
		Description: strings.TrimSpace(req.Description),
		Action:      strings.TrimSpace(req.Action),
		Owner:       strings.TrimSpace(req.Owner),
		OccurredAt:  req.OccurredAt,
		ResolvedAt:  req.ResolvedAt,
	}
	if incident.Title == "" {
		writeError(c, http.StatusBadRequest, "invalid_incident", "äº‹ä»¶æ ‡é¢˜ä¸èƒ½ä¸ºç©º")
		return
	}
	if err := a.db.Create(&incident).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "algorithm_incident_create_failed", "åº”æ€¥äº‹ä»¶åˆ›å»ºå¤±è´¥")
		return
	}
	a.writeAdminAudit(c, "algorithm_incident.create", "algorithm_incident", incident.ID, gin.H{"stb‡GGå7FGW47&VFVBÂ–æ6–FVçB§Ð ¦gVæ2†¤’Vç7W&TFVfVÇDÆv÷&—F†ÔF—66Æ÷7W&R‚’„Æv÷&—F†ÔF—66Æ÷7W&RÂW'&÷"’° —f"F—66Æ÷7W&RÆv÷&—F†ÔF—66Æ÷7W&P –W'"£ÒæF"ä÷&FW"‚&–B62"’äf—'7B‚fF—66Æ÷7W&R’äW'&÷  ––bW'"ÓÒæ–Â° —&WGW&âF—66Æ÷7W&RÂæ–À —Ð ––bv÷&Ô—5&V6÷&Dæ÷Df÷VæB†W'"’° —&WGW&âÆv÷&—F†ÔF—66Æ÷7W&W·ÒÂW'  —Ð –F—66Æ÷7W&RÒÆv÷&—F†ÔF—66Æ÷7W&W° ”Æv÷&—F†ÔæÖS¢.y›Þ™ÉnX[Kª¾Y»îx˜~yIþh‰YŽh‰iÈÞXªzé~k9R"À ”Æv÷&—F†ÕG—S¢.yIþh‰YŽh‰{²"À •6W'f–6TFW67&—F–öã¢.™Ú.Y	yJŽh‹~hùKé¾ih~yIþY»î8Xø.ˆ>Y»îYŽh‰8[˜:Ž{Én‹é8ˆxZ~x˜~KúîZHÞY(Îy»ŽXhÎ{¾Y»îx˜~yIþh‰YŽh‰iÈÞXª8""À •&÷f–FW$FW67&—F–öã¢.[›>Xû‹>yJŽzÊÎKˆžikžyIþh‰jŠYè¾ûÈÎKˆÞˆz®ŠÎŠêÞ{¸>Yû®zyIþh‰jŠYè¾ûÉ¾[›>Xû‹Iþ‹J>‹é>XZ^‹é>X{®k+¾yn8j~Šøn8iz^[ù~Y(ÎyJŽh‹~iØ>y¸®KùÞhªN8""À ”v÷fW&ææ6U7VÖÖ'“¢.yIþh‰X˜Þih~iÊÎZêjŽ8Xø.ˆ>Y»îZêjŽ8yIþh‰{¹>iéÎZêjŽ8XZÎ[ÈXˆnKª¾ZHÞjŽ8K«®[z^ZêjŽY(ÎK‹îhª^ZHN{Úî[Ú.h‰™zÞxêþ8""À ”Ö&¶–æu7VÖÖ'“¢.yIþh‰þ{Én‹éY»îx˜~YÊŽš(NŠxŽ8Kˆ¾‹ÛÞY(ÎXZÎ[ÈXˆnKª¾š^[^zK¢’yIþh‰j~ŠønûÈÎ[›n˜	®‹ørvVæW&F–öå÷&V6÷&Eö–BX[>ˆN‹ûÞkªþ8""À •W6W%&–v‡G57VÖÖ'“¢.yJŽh‹~Xúþ˜	®‹ø~XØþŠêî8™©zxiKþzÙn8zé~k9^XZÎzK®Y(ÎXh^ZëžK‹îhª^XZ^Xú>K¨nŠz>ŠxNX‰ž[›nhùKªNh©^Šøž8""À ”F—66Æ÷7W&UfW'6–öã¢###bÓbÓB"À •7FGW3¢Æv÷&—F†ÔF—66Æ÷7W&U7FGW4G&gBÀ —Ð ––bW'"£ÒæF"ä7&VFR‚fF—66Æ÷7W&R’äW'&÷#²W'"Òæ–Â° —&WGW&âÆv÷&—F†ÔF—66Æ÷7W&W·ÒÂW'  —Ð —&WGW&âF—66Æ÷7W&RÂæ–À§Ð ¦gVæ26ö×Æ–æ6UvR†2¦v–âä6öçFW‡B’†–çBÂ–çB’° —vR£ÒÖ„–çB†vWEVW'”–çB†2Â'vR"Â’Â —vU6—¦R£ÒÖ–ä–çB†Ö„–çB†vWEVW'”–çB†2Â'vU÷6—¦R"Â#’Â’Â —&WGW&âvRÂvU6—¦P§Ð ¦gVæ2—46öçFVçE6fWG•7FGW2‡7FGW27G&–ær’&ööÂ° —7v—F6‚7FGW2° –66R6öçFVçE6fWG•7FGW5VæF–ærÂ6öçFVçE6fWG•7FGW572Â6öçFVçE6fWG•7FGW5&V¦V7BÂ6öçFVçE6fWG•7FGW4ÖçVÅ&Wf–Ws  —&WGW&âG'VP –FVfVÇC  —&WGW&âfÇ6P —Ð§Ð ¦gVæ2†¤’v÷&´†4&Æö6¶–æt6öçFVçE&Wf–Wr‡v÷&²v÷&²’†&ööÂÂW'&÷"’° —7FGW6W2£Òµ×7G&–æw° ”6öçFVçE6fWG•7FGW5VæF–ærÀ ”6öçFVçE6fWG•7FGW4ÖçVÅ&Wf–WrÀ ”6öçFVçE6fWG•7FGW5&V¦V7BÀ —Ð —VW'’£ÒæF"äÖöFVÂ‚d6öçFVçE6fWG•&Wf–Ww·Ò’åv†W&R‚'7FGW2”âò"Â7FGW6W2 —VW'’ÒVW'’åv†W&R€ ’'v÷&µö–BÒòõ"‡F&vWE÷G—RÒòäBF&vWEö–BÒò’õ"vVæW&F–öå÷&V6÷&Eö–BÒò"À —v÷&²ä”BÀ ’'v÷&²"À —v÷&²ä”BÀ —v÷&²ävVæW&F–öå&V6÷&D”BÀ ’ —f"6÷VçB–çCc@ ––bW'"£ÒVW'’ä6÷VçB‚f6÷VçB’äW'&÷#²W'"Òæ–Â° —&WGW&âfÇ6RÂW'  —Ð —&WGW&â6÷VçBâÂæ–À§Ð ¦gVæ2v÷&Ô—5&V6÷&Dæ÷Df÷VæB†W'"W'&÷"’&ööÂ° —&WGW&âW'"ÓÒv÷&ÒäW'%&V6÷&Dæ÷Df÷Væ@§Ð  ¦gVæ2W66TÆ–¶R‡27G&–ær’7G&–ær° —2Ò7G&–æw2å&WÆ6TÆÂ‡2Â%Â"Â%ÅÂ" —2Ò7G&–æw2å&WÆ6TÆÂ‡2Â"R"Â%ÂR" —2Ò7G&–æw2å&WÆ6TÆÂ‡2Â%ò"Â%Åò" —&WGW&â0§Ð