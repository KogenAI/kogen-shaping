Feature: Keep the reason in bounded refusal records
  This feature covers only the recorded problem-line transformation, not gate verdicts
  or the uncut feedback sent to the shaper. Repeat counts below count Unicode runes.
  The supplied AoR observation counted 845 submission_refused problem lines across
  120 shaping-events.ndjson files under aor-climb/hypotheses/*/cells/*/run/cells/*/r*/*/;
  568 (67%) were exactly 300 characters. The reported example was attempt 3, gate
  contract_review, in h13-review-opus-high-vs-baseline-30cf6619/cells/rails-ft-data-export-and-import__review-opus-high.
  The first case preserves its supplied fragment and pads it to 600 runes with an
  explicitly synthetic why, since the lost historical reason is unknown. The reported
  stdout.txt ending "... Minitest::Reporters::Progres" is historical evidence of the
  same symptom, not a new input format. The narrow fix is in shape/checkorder.go;
  contract and decision line producers in shape/contract.go and shape/decision.go stay unchanged.

  The requested reason notation is | why: ...
  The supplied recorded fragment, verbatim, is:
  - Importing an account export creates a new account with everything | expectation: features/steps/rails-ft-data-export-and-import__driver/export_import_test.rb: get "#{@slug}/account/imports/new" | contract: config/environments/test.rb: config.action_mailer.default_url_options = { host: "example.com 

  Scenario: Preserve why while retaining all existing bounds
    Given shape-refusal-lines-keep-why-when-cut: these refusal transformation cases
      """
      [
        {
          "name": "600-rune contract, reported prefix and synthetic short reason",
          "input": [
            {
              "parts": [
                {
                  "text": "- Importing an account export creates a new account with everything | expectation: features/steps/rails-ft-data-export-and-import__driver/export_import_test.rb: get \"#{@slug}/account/imports/new\" | contract: config/environments/test.rb: config.action_mailer.default_url_options = { host: \"example.com "
                },
                {
                  "text": "界",
                  "repeat": 247
                },
                {
                  "text": " | why: importing must preserve the exported account"
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "inputPrefix": 243
                },
                {
                  "text": " ... "
                },
                {
                  "text": " | why: importing must preserve the exported account"
                }
              ]
            }
          ]
        },
        {
          "name": "600-rune decision with short reason",
          "input": [
            {
              "parts": [
                {
                  "text": "statement: "
                },
                {
                  "text": "界",
                  "repeat": 574
                },
                {
                  "text": " | why: because"
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "inputPrefix": 280
                },
                {
                  "text": " ... "
                },
                {
                  "text": " | why: because"
                }
              ]
            }
          ]
        },
        {
          "name": "long line without why retains original first 300 runes",
          "input": [
            {
              "parts": [
                {
                  "text": "界",
                  "repeat": 600
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "界",
                  "repeat": 300
                }
              ]
            }
          ]
        },
        {
          "name": "299-rune line with why",
          "input": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 290
                },
                {
                  "text": " | why: w"
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 290
                },
                {
                  "text": " | why: w"
                }
              ]
            }
          ]
        },
        {
          "name": "300-rune line with why",
          "input": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 291
                },
                {
                  "text": " | why: w"
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 291
                },
                {
                  "text": " | why: w"
                }
              ]
            }
          ]
        },
        {
          "name": "301-rune line with why",
          "input": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 292
                },
                {
                  "text": " | why: w"
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 286
                },
                {
                  "text": " ...  | why: w"
                }
              ]
            }
          ]
        },
        {
          "name": "149-rune why segment in a 600-rune line",
          "input": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 451
                },
                {
                  "text": " | why: "
                },
                {
                  "text": "理",
                  "repeat": 141
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 146
                },
                {
                  "text": " ...  | why: "
                },
                {
                  "text": "理",
                  "repeat": 141
                }
              ]
            }
          ]
        },
        {
          "name": "150-rune why segment in a 600-rune line",
          "input": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 450
                },
                {
                  "text": " | why: "
                },
                {
                  "text": "理",
                  "repeat": 142
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 145
                },
                {
                  "text": " ...  | why: "
                },
                {
                  "text": "理",
                  "repeat": 142
                }
              ]
            }
          ]
        },
        {
          "name": "151-rune why segment in a 600-rune line",
          "input": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 449
                },
                {
                  "text": " | why: "
                },
                {
                  "text": "理",
                  "repeat": 143
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "頭",
                  "repeat": 145
                },
                {
                  "text": " ...  | why: "
                },
                {
                  "text": "理",
                  "repeat": 142
                }
              ]
            }
          ]
        },
        {
          "name": "19 problem lines retain the existing ordered 20-line limit",
          "input": [
            {
              "parts": [
                {
                  "text": "earlier"
                }
              ],
              "copies": 19
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "earlier"
                }
              ],
              "copies": 19
            }
          ]
        },
        {
          "name": "20 problem lines retain the existing ordered 20-line limit",
          "input": [
            {
              "parts": [
                {
                  "text": "earlier"
                }
              ],
              "copies": 19
            },
            {
              "parts": [
                {
                  "text": "twentieth"
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "earlier"
                }
              ],
              "copies": 19
            },
            {
              "parts": [
                {
                  "text": "twentieth"
                }
              ]
            }
          ]
        },
        {
          "name": "21 problem lines retain the existing ordered 20-line limit",
          "input": [
            {
              "parts": [
                {
                  "text": "earlier"
                }
              ],
              "copies": 19
            },
            {
              "parts": [
                {
                  "text": "twentieth"
                }
              ]
            },
            {
              "parts": [
                {
                  "text": "discarded twenty-first"
                }
              ]
            }
          ],
          "expected": [
            {
              "parts": [
                {
                  "text": "earlier"
                }
              ],
              "copies": 19
            },
            {
              "parts": [
                {
                  "text": "twentieth"
                }
              ]
            }
          ]
        }
      ]
      """
    When shape-refusal-lines-keep-why-when-cut: the real Go cutProblems function processes the lines
    Then shape-refusal-lines-keep-why-when-cut: every bounded line matches its expected runes
