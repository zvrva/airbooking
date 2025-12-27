set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT_DIR}/internal/pb"
PROTOC_IMAGE="${PROTOC_IMAGE:-namely/protoc-all:1.36_0}"

mkdir -p "${OUT_DIR}"

docker run --rm -v "${ROOT_DIR}":/defs -w /defs \
  "${PROTOC_IMAGE}" \
  -f api/models/booking.proto \
  -f api/models/flight.proto \
  -f api/flights_api/flights.proto \
  -f api/bookings_api/bookings.proto \
  -i api \
  -o internal/pb \
  -l go \
  --go-source-relative \
  --with-gateway \
  --with-openapiv2 \
  --openapiv2-opt=allow_merge=true,merge_file_name=airbooking
