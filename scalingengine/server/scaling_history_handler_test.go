package server_test

import (
	"context"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/fakes"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3/lagertest"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/apis/scalinghistory"
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScalingHistoryHandler", func() {
	var (
		scalingEngineDB              *fakes.FakeScalingEngineDB
		handler                      *ScalingHistoryHandler
		err                          error
		history1, history2, history3 *models.AppScalingHistory
		scalingHistoryParams         scalinghistory.V1AppsGUIDScalingHistoriesGetParams
		history                      *scalinghistory.History
	)

	history1 = &models.AppScalingHistory{
		AppId:        "an-app-id",
		Timestamp:    222,
		ScalingType:  models.ScalingTypeDynamic,
		Status:       models.ScalingStatusSucceeded,
		OldInstances: 2,
		NewInstances: 4,
		Reason:       "a reason",
	}

	history2 = &models.AppScalingHistory{
		AppId:        "an-app-id",
		Timestamp:    333,
		ScalingType:  models.ScalingTypeSchedule,
		Status:       models.ScalingStatusFailed,
		OldInstances: 2,
		NewInstances: 4,
		Reason:       "a reason",
		Message:      "a message",
		Error:        "an error",
	}

	history3 = &models.AppScalingHistory{
		AppId:        "an-app-id",
		Timestamp:    444,
		ScalingType:  models.ScalingTypeDynamic,
		Status:       models.ScalingStatusIgnored,
		OldInstances: 2,
		NewInstances: 4,
		Reason:       "a reason",
		Message:      "a message",
	}

	history1Entry := scalinghistory.HistoryEntry{

		Status:       scalinghistory.OptHistoryEntryStatus{Value: 2, Set: true},
		AppID:        scalinghistory.OptGUID{Value: "an-app-id", Set: true},
		Timestamp:    scalinghistory.OptInt{Value: 444, Set: true},
		ScalingType:  scalinghistory.OptHistoryEntryScalingType{Value: 0, Set: true},
		OldInstances: scalinghistory.OptInt64{Value: 2, Set: true},
		NewInstances: scalinghistory.OptInt64{Value: 4, Set: true},
		Reason:       scalinghistory.OptString{Value: "a reason", Set: true},
		Message:      scalinghistory.OptString{Value: "a message", Set: true},
		OneOf:        scalinghistory.NewHistoryIgnoreEntryHistoryEntrySum(scalinghistory.HistoryIgnoreEntry{IgnoreReason: scalinghistory.NewOptString("a message")}),
	}

	history2Entry := scalinghistory.HistoryEntry{
		Status:       scalinghistory.OptHistoryEntryStatus{Value: 1, Set: true},
		AppID:        scalinghistory.OptGUID{Value: "an-app-id", Set: true},
		Timestamp:    scalinghistory.OptInt{Value: 333, Set: true},
		ScalingType:  scalinghistory.OptHistoryEntryScalingType{Value: 1, Set: true},
		OldInstances: scalinghistory.OptInt64{Value: 2, Set: true},
		NewInstances: scalinghistory.OptInt64{Value: 4, Set: true},
		Reason:       scalinghistory.OptString{Value: "a reason", Set: true},
		Message:      scalinghistory.OptString{Value: "a message", Set: true},
		OneOf:        scalinghistory.NewHistoryErrorEntryHistoryEntrySum(scalinghistory.HistoryErrorEntry{Error: scalinghistory.NewOptString("an error")}),
	}
	history3Entry := scalinghistory.HistoryEntry{
		Status:       scalinghistory.OptHistoryEntryStatus{Value: 0, Set: true},
		AppID:        scalinghistory.OptGUID{Value: "an-app-id", Set: true},
		Timestamp:    scalinghistory.OptInt{Value: 222, Set: true},
		ScalingType:  scalinghistory.OptHistoryEntryScalingType{Value: 0, Set: true},
		OldInstances: scalinghistory.OptInt64{Value: 2, Set: true},
		NewInstances: scalinghistory.OptInt64{Value: 4, Set: true},
		Reason:       scalinghistory.OptString{Value: "a reason", Set: true},
		Message:      scalinghistory.OptString{Value: "", Set: true},
		OneOf:        scalinghistory.NewHistorySuccessEntryHistoryEntrySum(scalinghistory.HistorySuccessEntry{}),
	}
	BeforeEach(func() {
		logger := lagertest.NewTestLogger("scaling-handler-test")
		scalingEngineDB = &fakes.FakeScalingEngineDB{}
		handler, err = NewScalingHistoryHandler(logger, scalingEngineDB)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("V1AppsGUIDScalingHistoriesGet", func() {
		BeforeEach(func() {
			scalingHistoryParams = scalinghistory.V1AppsGUIDScalingHistoriesGetParams{
				GUID: "an-app-id",
			}
		})
		JustBeforeEach(func() {
			history, err = handler.V1AppsGUIDScalingHistoriesGet(context.TODO(), scalingHistoryParams)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when request query string is valid", func() {
			Context("when start, end, order and include parameter are all in parameter", func() {
				BeforeEach(func() {
					scalingHistoryParams.StartTime = scalinghistory.NewOptInt(123)
					scalingHistoryParams.EndTime = scalinghistory.NewOptInt(567)
					scalingHistoryParams.OrderDirection = scalinghistory.NewOptV1AppsGUIDScalingHistoriesGetOrderDirection(scalinghistory.V1AppsGUIDScalingHistoriesGetOrderDirectionDesc)
				})

				It("retrieves scaling histories from database with the given start and end time and order ", func() {
					ctx, appid, start, end, order, includeAll, page, resultsPerPage := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
					Expect(ctx).NotTo(BeNil())
					Expect(appid).To(Equal("an-app-id"))
					Expect(start).To(Equal(int64(123)))
					Expect(end).To(Equal(int64(567)))
					Expect(order).To(Equal(db.DESC))
					Expect(includeAll).To(BeFalse())
					Expect(page).To(Equal(1))
					Expect(resultsPerPage).To(Equal(50))
				})
			})

			Context("when query database succeeds", func() {
				BeforeEach(func() {
					scalingHistoryParams.StartTime = scalinghistory.NewOptInt(123)
					scalingHistoryParams.EndTime = scalinghistory.NewOptInt(567)
					scalingHistoryParams.OrderDirection = scalinghistory.NewOptV1AppsGUIDScalingHistoriesGetOrderDirection(scalinghistory.V1AppsGUIDScalingHistoriesGetOrderDirectionDesc)

					scalingEngineDB.RetrieveScalingHistoriesReturns([]*models.AppScalingHistory{history3, history2, history1}, nil)
				})

				It("returns the scaling histories ", func() {
					Expect(history.Resources).To(HaveLen(3))
					Expect(history.Resources[0]).To(Equal(history1Entry))
					Expect(history.Resources[1]).To(Equal(history2Entry))
					Expect(history.Resources[2]).To(Equal(history3Entry))
					Expect(history.PrevURL.IsSet()).To(BeFalse())
					Expect(history.NextURL.IsSet()).To(BeFalse())
				})
			})

			Context("when paginating", func() {
				BeforeEach(func() {
					scalingHistoryParams.StartTime = scalinghistory.NewOptInt(123)
					scalingHistoryParams.EndTime = scalinghistory.NewOptInt(567)
					scalingHistoryParams.OrderDirection = scalinghistory.NewOptV1AppsGUIDScalingHistoriesGetOrderDirection(scalinghistory.V1AppsGUIDScalingHistoriesGetOrderDirectionDesc)
					scalingHistoryParams.Page = scalinghistory.NewOptInt(2)
					scalingHistoryParams.ResultsPerPage = scalinghistory.NewOptInt(1)

					scalingEngineDB.CountScalingHistoriesReturns(3, nil)
					scalingEngineDB.RetrieveScalingHistoriesReturns([]*models.AppScalingHistory{history2}, nil)
				})

				It("correctly paginates the result", func() {
					By("counting the results", func() {
						Expect(scalingEngineDB.CountScalingHistoriesCallCount()).To(Equal(1))
						Expect(history.TotalResults.Value).To(Equal(int64(3)))
						Expect(history.TotalPages.Value).To(Equal(int64(3)))
					})
					By("linking to the surrounding pages", func() {
						Expect(history.Resources).To(HaveLen(1))
						Expect(history.Page.Value).To(Equal(int64(2)))
						Expect(history.PrevURL.IsSet()).To(BeTrue())
						Expect(history.PrevURL.Value.RawQuery).To(ContainSubstring("&page=1"))
						Expect(history.NextURL.IsSet()).To(BeTrue())
						Expect(history.NextURL.Value.RawQuery).To(ContainSubstring("&page=3"))
					})
				})
			})

			Context("when paginating getting the deprecated order parameter", func() {
				BeforeEach(func() {
					scalingHistoryParams.StartTime = scalinghistory.NewOptInt(123)
					scalingHistoryParams.EndTime = scalinghistory.NewOptInt(567)
					//nolint:staticcheck // testing backwards-compatibility with our CF CLI plugin
					scalingHistoryParams.Order = scalinghistory.NewOptV1AppsGUIDScalingHistoriesGetOrder(scalinghistory.V1AppsGUIDScalingHistoriesGetOrderAsc)
					scalingHistoryParams.Page = scalinghistory.NewOptInt(2)
					scalingHistoryParams.ResultsPerPage = scalinghistory.NewOptInt(1)

					scalingEngineDB.CountScalingHistoriesReturns(3, nil)
					scalingEngineDB.RetrieveScalingHistoriesReturns([]*models.AppScalingHistory{history2}, nil)
				})

				It("correctly paginates the result", func() {
					By("counting the results", func() {
						Expect(scalingEngineDB.CountScalingHistoriesCallCount()).To(Equal(1))
						Expect(history.TotalResults.Value).To(Equal(int64(3)))
						Expect(history.TotalPages.Value).To(Equal(int64(3)))
					})
					By("forwarding the direction parameter to the DB", func() {
						ctx, appid, start, end, order, includeAll, page, resultsPerPage := scalingEngineDB.RetrieveScalingHistoriesArgsForCall(0)
						Expect(ctx).NotTo(BeNil())
						Expect(appid).To(Equal("an-app-id"))
						Expect(start).To(Equal(int64(123)))
						Expect(end).To(Equal(int64(567)))
						Expect(order).To(Equal(db.ASC))
						Expect(includeAll).To(BeFalse())
						Expect(page).To(Equal(2))
						Expect(resultsPerPage).To(Equal(1))
					})
					By("linking to the surrounding pages using the new orderDirection", func() {
						Expect(history.Resources).To(HaveLen(1))
						Expect(history.Page.Value).To(Equal(int64(2)))
						Expect(history.PrevURL.IsSet()).To(BeTrue())
						Expect(history.PrevURL.Value.RawQuery).To(ContainSubstring("&page=1"))
						Expect(history.PrevURL.Value.RawQuery).To(ContainSubstring("&order-direction=asc"))
						Expect(history.NextURL.IsSet()).To(BeTrue())
						Expect(history.NextURL.Value.RawQuery).To(ContainSubstring("&page=3"))
						Expect(history.NextURL.Value.RawQuery).To(ContainSubstring("&order-direction=asc"))
					})
				})
			})
		})
	})
})
