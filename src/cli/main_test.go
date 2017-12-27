package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	. "autoscaler/models"
	"cli/ui"
	cjson "cli/util/json"

	"code.cloudfoundry.org/cli/plugin/models"
	"code.cloudfoundry.org/cli/util/testhelpers/rpcserver"
	"code.cloudfoundry.org/cli/util/testhelpers/rpcserver/rpcserverfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("App-AutoScaler Commands", func() {
	const (
		fakeAppName     string = "fakeAppName"
		fakeAppId       string = "fakeAppId"
		fakeAccessToken string = "fakeAccessToken"
		policyFile      string = "policy.json"
	)

	var (
		rpcHandlers *rpcserverfakes.FakeHandlers
		ts          *rpcserver.TestServer
		apiServer   *ghttp.Server
		apiEndpoint string
		args        []string
		session     *gexec.Session
		err         error

		fakePolicy ScalingPolicy = ScalingPolicy{
			InstanceMin: 1,
			InstanceMax: 2,
			ScalingRules: []*ScalingRule{
				{
					MetricType:            "memoryused",
					StatWindowSeconds:     300,
					BreachDurationSeconds: 600,
					Threshold:             30,
					Operator:              "<=",
					CoolDownSeconds:       300,
					Adjustment:            "-1",
				},
			},
		}
	)

	BeforeEach(func() {

		os.Unsetenv("CF_TRACE")
		os.Setenv("AUTOSCALER_CONFIG_FILE", "test_config.json")

		//start rpc server to test cf cli plugin
		rpcHandlers = new(rpcserverfakes.FakeHandlers)

		//set rpc.CallCoreCommand to a successful call
		//rpc.CallCoreCommand is used in both cliConnection.CliCommand() and
		//cliConnection.CliWithoutTerminalOutput()
		rpcHandlers.CallCoreCommandStub = func(_ []string, retVal *bool) error {
			*retVal = true
			return nil
		}

		//set rpc.GetOutputAndReset to return empty string; this is used by CliCommand()/CliWithoutTerminalOutput()
		rpcHandlers.GetOutputAndResetStub = func(_ bool, retVal *[]string) error {
			*retVal = []string{"{}"}
			return nil
		}

		ts, err = rpcserver.NewTestRPCServer(rpcHandlers)
		Expect(err).NotTo(HaveOccurred())

		err = ts.Start()
		Expect(err).NotTo(HaveOccurred())

		//start fake AutoScaler API server
		apiServer = ghttp.NewServer()
		apiServer.RouteToHandler("GET", "/health",
			ghttp.RespondWith(http.StatusOK, ""),
		)

		apiEndpoint = apiServer.URL()

	})

	AfterEach(func() {
		ts.Stop()
		apiServer.Close()
		if _, err = os.Stat(policyFile); !os.IsNotExist(err) {
			err = os.Remove(policyFile)
		}

	})

	Describe("Commands autoscaling-api, asa", func() {

		Context("Set api endpoint", func() {

			BeforeEach(func() {
				args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
			})

			Context("with http server", func() {
				Context("Succeed", func() {
					It("to say 'Setting AutoScaler api endpoint to ...' ", func() {
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.SetAPIEndpoint, apiEndpoint))
						Expect(session.ExitCode()).To(Equal(0))
					})
				})

				Context("Failed when api server is unaccessible ", func() {
					BeforeEach(func() {
						apiServer.Close()
					})

					It("connection refused", func() {
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("connection refused"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("Failed when no /health endpoint ", func() {

					BeforeEach(func() {
						apiServer.RouteToHandler("GET", "/health",
							ghttp.RespondWith(http.StatusNotFound, ""),
						)
					})

					It("Invalid api endpoint", func() {
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say(ui.InvalidAPIEndpoint, apiEndpoint))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

			})

			Context("with a self-signed TLS server", func() {
				var (
					apiTLSServer   *ghttp.Server
					apiTLSEndpoint string
				)

				BeforeEach(func() {
					apiTLSServer = ghttp.NewTLSServer()

					apiTLSServer.RouteToHandler("GET", "/health",
						ghttp.RespondWith(http.StatusOK, ""),
					)
					apiTLSEndpoint = apiTLSServer.URL()
				})

				AfterEach(func() {
					apiTLSServer.Close()
				})

				It("require --skip-ssl-validation option", func() {
					args = []string{ts.Port(), "autoscaling-api", apiTLSEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.InvalidSSLCerts, apiTLSEndpoint))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("succeed with --skip-ssl-validation ", func() {
					args = []string{ts.Port(), "autoscaling-api", apiTLSEndpoint, "--skip-ssl-validation"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.SetAPIEndpoint, apiTLSEndpoint))
					Expect(session.ExitCode()).To(Equal(0))
				})

				It("attach 'https' as the default protocol when prefix is missing ", func() {
					args = []string{ts.Port(), "autoscaling-api", strings.TrimPrefix(apiTLSEndpoint, "https://"), "--skip-ssl-validation"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.SetAPIEndpoint, apiTLSEndpoint))
					Expect(session.ExitCode()).To(Equal(0))
				})
			})
		})

		Context("Unset api endpoint", func() {

			It("Succeed", func() {
				args = []string{ts.Port(), "autoscaling-api", "--unset"}
				session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				session.Wait()
				Expect(session).To(gbytes.Say(ui.UnsetAPIEndpoint))
				Expect(session.ExitCode()).To(Equal(0))
			})

			It("'unset'take higher proprity than the other argument", func() {
				args = []string{ts.Port(), "autoscaling-api", apiEndpoint, "--unset"}
				session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				session.Wait()
				Expect(session).To(gbytes.Say(ui.UnsetAPIEndpoint))
				Expect(session.ExitCode()).To(Equal(0))
			})
		})

		Context("Show api endpoint", func() {

			Context("No previous end-point setting", func() {

				BeforeEach(func() {
					args = []string{ts.Port(), "autoscaling-api", "--unset"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
				})

				It("response with no endpoint..", func() {
					args = []string{ts.Port(), "autoscaling-api"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.NoEndpoint))
					Expect(session.ExitCode()).To(Equal(0))
				})
			})

			Context("End-point was set", func() {

				BeforeEach(func() {
					args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
				})

				It("response with the pre-defined endpoint..", func() {
					args = []string{ts.Port(), "autoscaling-api"}
					session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say(ui.APIEndpoint, apiEndpoint))
					Expect(session.ExitCode()).To(Equal(0))
				})
			})

		})

	})

	Describe("Commands autoscaling-policy, asp", func() {

		Context("autoscaling-policy", func() {

			Context("when the args are not properly provided", func() {
				It("Require APP_NAME as argument", func() {
					args = []string{ts.Port(), "autoscaling-policy"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("required argument `APP_NAME` was not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf not login", func() {
				It("exits with 'You must be logged in' error ", func() {
					args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say("You must be logged in"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf login", func() {
				BeforeEach(func() {
					rpcHandlers.IsLoggedInStub = func(args string, retVal *bool) error {
						*retVal = true
						return nil
					}
				})

				Context("when app not found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							return errors.New("App fakeApp not found")
						}
					})

					It("exits with 'App not found' error ", func() {
						args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
						session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("App fakeApp not found"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("when the app is found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							*retVal = plugin_models.GetAppModel{
								Guid: fakeAppId,
							}
							return nil
						}
					})

					JustBeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					Context("when access token is wrong", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = "incorrectAccessToken"
								return nil
							}

							apiServer.RouteToHandler("GET", "/v1/apps/"+fakeAppId+"/policy",
								ghttp.RespondWith(http.StatusUnauthorized, ""),
							)
						})

						It("failed with 401 error", func() {
							args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
							session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()

							Expect(session).To(gbytes.Say("Failed to access AutoScaler API Endpoint"))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when access token is correct", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = fakeAccessToken
								return nil
							}
						})

						Context("when policy not found", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("GET", "/v1/apps/"+fakeAppId+"/policy",
									ghttp.RespondWith(http.StatusNotFound, ""),
								)
							})

							It("404 returned", func() {
								args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session).To(gbytes.Say(ui.PolicyNotFound, fakeAppName))
								Expect(session.ExitCode()).To(Equal(1))

							})
						})

						Context("when policy exist ", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("GET", "/v1/apps/"+fakeAppId+"/policy",
									ghttp.CombineHandlers(
										ghttp.RespondWithJSONEncoded(http.StatusOK, &fakePolicy),
										ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
									),
								)
							})

							It("Succeed to print the policy to stdout", func() {

								args = []string{ts.Port(), "autoscaling-policy", fakeAppName}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session.Out).To(gbytes.Say(ui.ShowPolicyHint, fakeAppName))
								Expect(session.Out).To(gbytes.Say("OK"))
								policy := bytes.TrimLeft(session.Out.Contents(), fmt.Sprintf(ui.ShowPolicyHint+"\nOK", fakeAppName))

								fmt.Println(string(policy))
								var actualPolicy ScalingPolicy
								_ = json.Unmarshal(policy, &actualPolicy)

								Expect(actualPolicy).To(MatchFields(IgnoreExtras, Fields{
									"InstanceMin": BeNumerically("==", fakePolicy.InstanceMin),
									"InstanceMax": BeNumerically("==", fakePolicy.InstanceMax),
								}))

								Expect(*actualPolicy.ScalingRules[0]).To(MatchFields(IgnoreExtras, Fields{
									"MetricType":            Equal(fakePolicy.ScalingRules[0].MetricType),
									"StatWindowSeconds":     BeNumerically("==", fakePolicy.ScalingRules[0].StatWindowSeconds),
									"BreachDurationSeconds": BeNumerically("==", fakePolicy.ScalingRules[0].BreachDurationSeconds),
									"Threshold":             BeNumerically("==", fakePolicy.ScalingRules[0].Threshold),
									"Operator":              Equal(fakePolicy.ScalingRules[0].Operator),
									"CoolDownSeconds":       BeNumerically("==", fakePolicy.ScalingRules[0].CoolDownSeconds),
									"Adjustment":            Equal(fakePolicy.ScalingRules[0].Adjustment),
								}))

							})

							Context("Succeed to print the policy to file", func() {

								It("Succeed", func() {
									args = []string{ts.Port(), "autoscaling-policy", fakeAppName, "--output", policyFile}
									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session).To(gbytes.Say("OK"))

									Expect(policyFile).To(BeARegularFile())
									contents, err := ioutil.ReadFile(policyFile)
									Expect(err).NotTo(HaveOccurred())

									var actualPolicy ScalingPolicy
									_ = json.Unmarshal(contents, &actualPolicy)

									Expect(actualPolicy).To(MatchFields(IgnoreExtras, Fields{
										"InstanceMin": BeNumerically("==", fakePolicy.InstanceMin),
										"InstanceMax": BeNumerically("==", fakePolicy.InstanceMax),
									}))

									Expect(*actualPolicy.ScalingRules[0]).To(MatchFields(IgnoreExtras, Fields{
										"MetricType":            Equal(fakePolicy.ScalingRules[0].MetricType),
										"StatWindowSeconds":     BeNumerically("==", fakePolicy.ScalingRules[0].StatWindowSeconds),
										"BreachDurationSeconds": BeNumerically("==", fakePolicy.ScalingRules[0].BreachDurationSeconds),
										"Threshold":             BeNumerically("==", fakePolicy.ScalingRules[0].Threshold),
										"Operator":              Equal(fakePolicy.ScalingRules[0].Operator),
										"CoolDownSeconds":       BeNumerically("==", fakePolicy.ScalingRules[0].CoolDownSeconds),
										"Adjustment":            Equal(fakePolicy.ScalingRules[0].Adjustment),
									}))

								})

							})
						})

					})

				})

			})
		})
	})

	Describe("Commands attach-autoscaling-policy, aasp", func() {

		Context("attach-autoscaling-policy", func() {

			Context("when the args are not properly provided", func() {
				It("Require both APP_NAME and PATH_TO_POLICY_FILE as argument", func() {
					args = []string{ts.Port(), "attach-autoscaling-policy"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("the required arguments `APP_NAME` and `PATH_TO_POLICY_FILE` were not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})

				It("Require PATH_TO_POLICY_FILE as argument", func() {
					args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("the required argument `PATH_TO_POLICY_FILE` was not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf not login", func() {
				It("exits with 'You must be logged in' error ", func() {
					args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say("You must be logged in"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf login", func() {
				BeforeEach(func() {
					rpcHandlers.IsLoggedInStub = func(args string, retVal *bool) error {
						*retVal = true
						return nil
					}
				})

				Context("when app not found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							return errors.New("App fakeApp not found")
						}
					})

					It("exits with 'App not found' error ", func() {
						args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
						session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("App fakeApp not found"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("when the app is found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							*retVal = plugin_models.GetAppModel{
								Guid: fakeAppId,
							}
							return nil
						}
					})

					JustBeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					Context("when policy file is not exist", func() {
						BeforeEach(func() {
							err = os.Remove(policyFile)
						})

						It("Failed when policy file not exist", func() {
							args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
							session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()
							Expect(session).To(gbytes.Say(ui.FailToLoadPolicyFile, policyFile))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when policy file is empty", func() {
						BeforeEach(func() {
							err = ioutil.WriteFile(policyFile, nil, 0666)
							Expect(err).NotTo(HaveOccurred())
						})

						It("Failed when policy file is empty", func() {
							args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
							session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()
							Expect(session).To(gbytes.Say(strings.TrimSuffix(ui.InvalidPolicy, "%v.")))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when policy file is invalid json", func() {
						BeforeEach(func() {
							invalidPolicy := []byte(`{"policy":invalidPolicy}`)
							err = ioutil.WriteFile(policyFile, invalidPolicy, 0666)
							Expect(err).NotTo(HaveOccurred())
						})

						It("Failed when policy file is empty", func() {
							args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
							session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()
							Expect(session).To(gbytes.Say(strings.TrimSuffix(ui.InvalidPolicy, "%v.")))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when both app & policy is written in json format correctly", func() {

						BeforeEach(func() {
							policyBytes, err := cjson.MarshalWithoutHTMLEscape(fakePolicy)
							Expect(err).NotTo(HaveOccurred())
							err = ioutil.WriteFile(policyFile, policyBytes, 0666)
							Expect(err).NotTo(HaveOccurred())
						})

						Context("when access token is wrong", func() {
							BeforeEach(func() {
								rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
									*retVal = "incorrectAccessToken"
									return nil
								}

								apiServer.RouteToHandler("PUT", "/v1/apps/"+fakeAppId+"/policy",
									ghttp.RespondWith(http.StatusUnauthorized, ""),
								)
							})

							It("failed with 401 error", func() {
								args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session).To(gbytes.Say("Failed to access AutoScaler API Endpoint"))
								Expect(session.ExitCode()).To(Equal(1))
							})
						})

						Context("when access token is correct", func() {
							BeforeEach(func() {
								rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
									*retVal = fakeAccessToken
									return nil
								}
							})

							Context("when No policy defined previously", func() {
								BeforeEach(func() {
									apiServer.RouteToHandler("PUT", "/v1/apps/"+fakeAppId+"/policy",
										ghttp.CombineHandlers(
											ghttp.RespondWith(http.StatusCreated, ""),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)
								})

								It("Succeed with 201", func() {
									args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.AttachPolicyHint, fakeAppName))
									Expect(session.Out).To(gbytes.Say("OK"))

								})
							})

							Context("when policy exist previously ", func() {
								BeforeEach(func() {
									apiServer.RouteToHandler("PUT", "/v1/apps/"+fakeAppId+"/policy",
										ghttp.CombineHandlers(
											ghttp.RespondWith(http.StatusOK, ""),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)
								})

								It("Succeed with 200", func() {

									args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.AttachPolicyHint, fakeAppName))
									Expect(session.Out).To(gbytes.Say("OK"))

								})

							})

							Context("when attached policy definition is invalid ", func() {
								BeforeEach(func() {
									apiServer.RouteToHandler("PUT", "/v1/apps/"+fakeAppId+"/policy",
										ghttp.CombineHandlers(
											ghttp.RespondWith(http.StatusBadRequest, `{"success":false,"error":[{"property":"instance_min_count","message":"instance_min_count and instance_max_count values are not compatible","instance":{"instance_max_count":2,"instance_min_count":10,"scaling_rules":[{"adjustment":"+1","breach_duration_secs":600,"cool_down_secs":300,"metric_type":"memoryused","operator":">","stat_window_secs":300,"threshold":100},{"adjustment":"-1","breach_duration_secs":600,"cool_down_secs":300,"metric_type":"memoryused","operator":"<=","stat_window_secs":300,"threshold":5}]},"stack":"instance_min_count 10 is higher or equal to instance_max_count 2 in policy_json"}],"result":null}`),
											ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
										),
									)

									fakePolicy.InstanceMin = 10
									fakePolicy.InstanceMax = 2
									policyBytes, err := cjson.MarshalWithoutHTMLEscape(fakePolicy)
									Expect(err).NotTo(HaveOccurred())
									err = ioutil.WriteFile(policyFile, policyBytes, 0666)
									Expect(err).NotTo(HaveOccurred())

								})

								It("Failed with 400", func() {

									args = []string{ts.Port(), "attach-autoscaling-policy", fakeAppName, policyFile}
									session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
									Expect(err).NotTo(HaveOccurred())
									session.Wait()

									Expect(session.Out).To(gbytes.Say(ui.AttachPolicyHint, fakeAppName))
									Expect(session).To(gbytes.Say("FAILED"))
									Expect(session).To(gbytes.Say(ui.InvalidPolicy, "instance_min_count 10 is higher or equal to instance_max_count 2 in policy_json"))
									Expect(session.ExitCode()).To(Equal(1))

								})

							})

						})
					})

				})

			})
		})
	})

	Describe("Commands detach-autoscaling-policy, dasp", func() {

		Context("detach-autoscaling-policy", func() {

			Context("when the args are not properly provided", func() {
				It("Require APP_NAME as argument", func() {
					args = []string{ts.Port(), "detach-autoscaling-policy"}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()

					Expect(session).To(gbytes.Say("required argument `APP_NAME` was not provided"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf not login", func() {
				It("exits with 'You must be logged in' error ", func() {
					args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
					session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					session.Wait()
					Expect(session).To(gbytes.Say("You must be logged in"))
					Expect(session.ExitCode()).To(Equal(1))
				})
			})

			Context("when cf login", func() {
				BeforeEach(func() {
					rpcHandlers.IsLoggedInStub = func(args string, retVal *bool) error {
						*retVal = true
						return nil
					}
				})

				Context("when app not found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							return errors.New("App fakeApp not found")
						}
					})

					It("exits with 'App not found' error ", func() {
						args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
						session, err := gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
						Expect(session).To(gbytes.Say("App fakeApp not found"))
						Expect(session.ExitCode()).To(Equal(1))
					})
				})

				Context("when the app is found", func() {
					BeforeEach(func() {
						rpcHandlers.GetAppStub = func(_ string, retVal *plugin_models.GetAppModel) error {
							*retVal = plugin_models.GetAppModel{
								Guid: fakeAppId,
							}
							return nil
						}
					})

					JustBeforeEach(func() {
						args = []string{ts.Port(), "autoscaling-api", apiEndpoint}
						session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						session.Wait()
					})

					Context("when access token is wrong", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = "incorrectAccessToken"
								return nil
							}

							apiServer.RouteToHandler("DELETE", "/v1/apps/"+fakeAppId+"/policy",
								ghttp.RespondWith(http.StatusUnauthorized, ""),
							)
						})

						It("failed with 401 error", func() {
							args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
							session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
							Expect(err).NotTo(HaveOccurred())
							session.Wait()

							Expect(session).To(gbytes.Say("Failed to access AutoScaler API Endpoint"))
							Expect(session.ExitCode()).To(Equal(1))
						})
					})

					Context("when access token is correct", func() {
						BeforeEach(func() {
							rpcHandlers.AccessTokenStub = func(args string, retVal *string) error {
								*retVal = fakeAccessToken
								return nil
							}
						})

						Context("when policy not found", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("DELETE", "/v1/apps/"+fakeAppId+"/policy",
									ghttp.RespondWith(http.StatusNotFound, ""),
								)
							})

							It("404 returned", func() {
								args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session.Out).To(gbytes.Say(ui.DetachPolicyHint, fakeAppName))
								Expect(session).To(gbytes.Say(ui.PolicyNotFound, fakeAppName))
								Expect(session.ExitCode()).To(Equal(1))

							})
						})

						Context("when policy exist ", func() {
							BeforeEach(func() {
								apiServer.RouteToHandler("DELETE", "/v1/apps/"+fakeAppId+"/policy",
									ghttp.CombineHandlers(
										ghttp.RespondWith(http.StatusOK, ""),
										ghttp.VerifyHeaderKV("Authorization", fakeAccessToken),
									),
								)
							})

							It("Succeed", func() {

								args = []string{ts.Port(), "detach-autoscaling-policy", fakeAppName}
								session, err = gexec.Start(exec.Command(validPluginPath, args...), GinkgoWriter, GinkgoWriter)
								Expect(err).NotTo(HaveOccurred())
								session.Wait()

								Expect(session.Out).To(gbytes.Say(ui.DetachPolicyHint, fakeAppName))
								Expect(session.Out).To(gbytes.Say("OK"))
							})

						})

					})

				})

			})
		})
	})

})
