# confluence-cli

`confluence-cli`는 `confluence.example.com` 같은 **Self-hosted Confluence (Server/Data Center)** 환경에서
AI Agent가 문서를 읽고/쓰기 위한 안정적인 인터페이스를 제공하기 위해 만듭니다.

> **상태: 초기 동작 버전 (alpha).**
> Go로 구현된 CLI가 동작합니다: `search` / `get` / `create` / `update` / `comment`.
> 표준 라이브러리만 사용하며(외부 의존성 0), 단일 바이너리로 크로스플랫폼 배포가 가능합니다.

## 왜 이 프로젝트가 필요한가

Atlassian 공식 Remote MCP 서버(`mcp.atlassian.com`, Rovo MCP)는 **Confluence Cloud 전용**이며,
**Server/Data Center 환경은 직접 지원하지 않습니다.**
공식 MCP는 Cloud 중심 워크플로우(OAuth, Cloud REST API base)에 맞춰져 있어,
셀프 호스팅 환경에서는 인증 방식·네트워크 제약·권한 모델이 다릅니다.

이 때문에 다음이 필요했습니다.

- AI Agent가 내부 Confluence를 직접 호출할 수 있는 **전용 CLI 계층**
- API 토큰/인증 방식, 기본 파라미터, 오류 응답을 일괄 처리하는 래퍼
- Skill이 필요한 순간에만 도구 정보를 불러올 수 있게 하는 의도적인 경량화
- 추후 로그·감사 기준에 맞는 실행 기록/오류 추적

> 참고: 셀프 호스팅 Atlassian을 지원하는 비공식 MCP(예: `sooperset/mcp-atlassian`)가 존재하지만,
> 공식 지원/감사 추적/사내 권한 모델 관점에서 자체 CLI 계층을 두는 것이 이 프로젝트의 선택입니다.

## 프로젝트 목표 (초기)

1. Confluence 문서 검색/조회/생성/수정 명령을 표준화된 CLI 서브커맨드로 제공
2. AI/자동화가 바로 사용할 수 있는 JSON/텍스트 출력 제공
3. 인증(사이트 URL, 사용자, 토큰/비밀번호)을 환경변수/설정 파일로 분리
4. 실패 시 명확한 에러 메시지와 재시도 전략 제공
5. 최소 의존성으로 운영 환경 안전성 확보

## 핵심 기능

구현 완료:

- `confluence-cli search` : 키워드·제목·스페이스·CQL로 페이지 검색
- `confluence-cli get` : 페이지 조회 (storage 본문 출력 옵션 포함)
- `confluence-cli create` : 새 페이지 생성 (storage / wiki representation, 부모 지정)
- `confluence-cli update` : 기존 페이지 업데이트 (버전 자동 증가)
- `confluence-cli comment` : 페이지에 댓글 추가
- `confluence-cli generate-skill` : AI 에이전트용 `confluence-skill.md` 생성 (claude/codex/gemini/opencode/generic)

계획 중 (로드맵 참고):

- `confluence-cli diff` : 버전 비교 / 마크다운 변환 도움
- `confluence-cli attachment` : 첨부 조회·다운로드

## 설치

소개 페이지: <https://jhl-labs.github.io/confluence-cli>

```bash
# 스크립트 설치 (Linux/macOS) — 최신 릴리스 바이너리를 받습니다
curl -fsSL https://jhl-labs.github.io/confluence-cli/install.sh | sh
```

또는 [Releases](https://github.com/jhl-labs/confluence-cli/releases/latest)에서
플랫폼별 단일 바이너리를 직접 내려받을 수 있습니다 (Linux/macOS amd64·arm64, Windows amd64).

## 소스에서 빌드

Go 1.26+ 환경이 필요합니다.

```bash
# 빌드 (현재 플랫폼)
make build          # ./confluence-cli 생성
# 또는
go build -o confluence-cli .

# 전체 플랫폼 릴리즈 바이너리 (dist/)
make dist           # linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64

# 테스트
make test
```

크로스 컴파일은 Go 표준 방식 그대로 동작합니다.

```bash
GOOS=windows GOARCH=amd64 go build -o confluence-cli.exe .
GOOS=darwin  GOARCH=arm64 go build -o confluence-cli .
```

## 사용법

```bash
# 검색 (텍스트 / 스페이스 / 원시 CQL)
confluence-cli search --text "release notes" --space ENG --limit 10
confluence-cli search --cql 'type=page AND title ~ "design"'

# 페이지 조회 (--output text 로 사람이 읽기 쉬운 요약, 기본은 JSON)
confluence-cli get --id 123456 --output text
confluence-cli get --id 123456 --body            # storage 본문만 출력

# 생성 (본문은 인자 / 파일 / stdin)
confluence-cli create --space ENG --title "New Page" --body "<p>Hello</p>"
confluence-cli create --space ENG --title "From file" --body-file page.xhtml
echo "<p>piped</p>" | confluence-cli create --space ENG --title "Piped" --body-file -

# 수정 (버전은 자동으로 현재+1)
confluence-cli update --id 123456 --body-file updated.xhtml

# 댓글
confluence-cli comment --id 123456 --body "<p>LGTM</p>"

# 에이전트용 스킬 문서 생성 (confluence-skill.md)
confluence-cli generate-skill                 # 범용(generic)
confluence-cli generate-skill claude          # Claude SKILL.md (프론트매터 포함)
confluence-cli generate-skill codex           # Codex AGENTS.md 형식
confluence-cli generate-skill gemini          # Gemini GEMINI.md 형식
confluence-cli generate-skill opencode        # opencode AGENTS.md 형식
confluence-cli generate-skill claude --stdout # 파일 대신 표준출력
```

`generate-skill`은 이 CLI의 사용법(인증·명령어·storage 포맷 주의사항)을 담은
`confluence-skill.md`를 생성합니다. 플랫폼별로 포맷이 다릅니다.

- 인자 없음 / `generic` : 프론트매터 없는 범용 마크다운
- `claude` : `name`/`description` YAML 프론트매터를 가진 Claude 스킬(`SKILL.md`로 사용)
- `codex` / `opencode` : `AGENTS.md` 컨벤션
- `gemini` : `GEMINI.md` 컨텍스트 파일 컨벤션

기본 출력 파일은 `confluence-skill.md`이며, `--out <path>`로 경로를, `--stdout`으로
표준출력을, `--force`로 덮어쓰기를 지정할 수 있습니다.

모든 명령은 기본적으로 **JSON**을 출력하므로 AI Agent/스크립트가 바로 파싱할 수 있고,
`--output text`로 사람이 읽기 쉬운 요약을 얻을 수 있습니다.

> 본문 형식: 기본은 `storage`(XHTML). 마크다운/위키 문법을 그대로 쓰고 싶다면
> `--representation wiki`로 보내면 Confluence 서버가 변환합니다. (아래 *본문 형식 주의* 참고)

## 설정 우선순위

낮음 → 높음 순으로 덮어씁니다: **설정 파일 < 환경변수 < 커맨드라인 플래그**

- 설정 파일: `$CONFLUENCE_CONFIG` 또는 `~/.config/confluence-cli/config.json`
  (Windows는 `%AppData%`, macOS는 `~/Library/Application Support` — `os.UserConfigDir` 기준)
- 예시는 [`config.example.json`](./config.example.json) 참고

## 인증 (Server/Data Center 기준)

Confluence **Server/Data Center**는 Cloud와 인증 방식이 다릅니다.
Cloud의 `이메일 + API 토큰` 방식이 아니라, 아래 둘 중 하나를 사용합니다.

- **Personal Access Token (권장)** — Confluence **7.9 이상**에서 지원.
  `Authorization: Bearer <token>` 헤더로 호출합니다.
  토큰은 Confluence UI의 *프로필 → Settings → Personal access tokens*에서 발급하며, 만료일 설정을 권장합니다.
- **Basic 인증** — `사용자명 + 비밀번호`(또는 토큰)를 Base64로 인코딩.
  PAT를 쓸 수 없는 구버전 호환용. 장기 운영에는 비권장.

```bash
# PAT 동작 확인 예시
curl -H "Authorization: Bearer $CONFLUENCE_TOKEN" \
  "$CONFLUENCE_BASE_URL/rest/api/content?limit=1"
```

### 환경변수 (예시)

| 변수 | 설명 | 예시 |
|---|---|---|
| `CONFLUENCE_BASE_URL` | 사이트 베이스 URL | `https://confluence.example.com` |
| `CONFLUENCE_TOKEN` | Personal Access Token | `NjA2M...` |
| `CONFLUENCE_USER` | Basic 인증 사용 시 사용자명 | `agent-bot` |
| `CONFLUENCE_PASSWORD` | Basic 인증 사용 시 비밀번호/토큰 | `••••••` |
| `CONFLUENCE_SPACE` | 기본 스페이스 키 (`search`/`create`에서 `--space` 생략 가능) | `INFRA` |

> 토큰/비밀번호는 셸 히스토리·로그에 남지 않도록 `.env`(gitignore) 또는 시크릿 매니저로 관리합니다.

## REST API 기준

- 베이스 경로: `{BASE_URL}/rest/api/...` (Server/DC REST API)
- 검색: `GET /rest/api/content/search?cql=...` — **CQL(Confluence Query Language)** 사용
- 조회: `GET /rest/api/content/{id}?expand=body.storage,version,space`
- 생성/수정: `POST` / `PUT /rest/api/content/{id}` — 수정 시 `version.number`를 +1로 올려 전송
- 공식 문서: <https://developer.atlassian.com/server/confluence/confluence-server-rest-api/>

### ⚠️ 본문 형식 주의: Confluence는 마크다운이 아니다

Confluence 본문은 마크다운이 아니라 **XHTML 기반 "storage format"**입니다.
AI Agent가 흔히 마크다운을 생성하므로, create/update 경로에는 **마크다운 → storage format 변환**이 필요합니다.

- 본문 작성/수정: `body.storage`(storage format, 정본)
- 읽기 전용 렌더 결과: `body.view`(렌더된 HTML, 쓰기에는 사용 불가)
- `diff`/변환 명령은 이 변환 계층을 보조하는 용도로 둡니다.

이 변환을 어떻게/어디까지 지원할지가 이 프로젝트의 난이도를 좌우하는 핵심 포인트입니다.

## 대상 사용자

- 내부 위키 운영자
- AI Agent/자동화 스크립트를 운영하는 플랫폼 팀
- 문서 표준화를 원하는 개발/운영 조직

## 기술 방향

- 언어는 **Go**로 확정. 단일 정적 바이너리 → 런타임 설치 없이 크로스플랫폼 배포가 쉬움
- **외부 의존성 0** (Go 표준 라이브러리만 사용) → 운영 환경 공급망 위험 최소화
- 인증은 **Confluence Server/Data Center가 지원하는 방식**(PAT Bearer / Basic)에 맞춰 구현
- 스키마/파라미터는 공식 REST API 응답을 기준으로 고정
- 배경: Confluence **Server는 2024년 2월 지원 종료(EOL)** 되었고, 현재 셀프호스팅 지원 라인은
  **Data Center**입니다. 본 CLI는 Data Center를 1차 타깃으로 삼습니다.

## 프로젝트 구조

```
.
├── main.go              # 엔트리포인트, 서브커맨드 디스패치
├── common.go            # 공통 플래그(인증/출력) + 클라이언트 생성
├── output.go            # JSON / text 출력
├── cmd_*.go             # search / get / create / update / comment
├── internal/
│   ├── config/          # 설정 로딩 (파일 < 환경변수 < 플래그)
│   └── confluence/      # REST 클라이언트 (auth, 재시도, 에러), 콘텐츠 API
└── Makefile             # build / test / dist(크로스컴파일)
```

## 향후 산출물

- `docs/`: 사용법, 인증, 트러블슈팅
- `examples/`: 실제 호출 예시
- `tests/`: API 모의(Mock) 기반 단위 테스트
- `scripts/` 또는 `bin/`: 배포·릴리즈 보조 스크립트

## 로드맵

- [x] 언어/런타임 확정(Go) 및 프로젝트 스캐폴딩
- [x] 인증 계층 (PAT / Basic) + 설정 로딩(파일/환경변수/플래그)
- [x] `get` / `search`(CQL) 읽기 명령
- [x] `create` / `update` 쓰기 명령 (storage / wiki representation)
- [x] JSON 출력 표준화 + 에러/재시도(429·5xx, Retry-After) 전략
- [x] httptest 기반 단위 테스트
- [ ] `comment` 조회/삭제, `attachment` 다운로드, `diff`(버전 비교)
- [ ] 마크다운 → storage format 변환 계층
- [ ] 페이지네이션(`--start`/cursor) 및 대량 처리
- [ ] `docs/`, `examples/` 문서 보강, 릴리즈 자동화(CI)

## 운영 원칙

- 내부 토큰/패스워드는 절대 코드나 로그에 노출하지 않음
- 문서/업데이트는 최소 권한 원칙으로 수행
- 작업 단위는 명령형, 응답은 AI가 소비하기 쉬운 구조로 통일

## 라이선스

[MIT](./LICENSE)
