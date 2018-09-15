bookkeeper:
  build: .
  restart: always
  volumes:
    - .:/opt/app
  working_dir: /opt/app
  command: ["./bookkeeper",
    "-token", "{{ secret "telegram/bot_token_bookkeeper" }}",
    "-users", "Sirius={{ secret "telegram/user_id_sirius" }},Jian={{ secret "telegram/user_id_jian" }}",
    "-admin", "Sirius",
    "-sheet", "{{ secret "google/spreadsheet_id_bookkeeper" }}"]
  log_opt:
    max-size: "50m"
