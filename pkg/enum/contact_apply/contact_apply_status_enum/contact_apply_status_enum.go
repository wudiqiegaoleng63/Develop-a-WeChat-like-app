package contact_apply_status_enum

const (
    PENDING int8 = 0  // 申请中 - 等待处理
    AGREE   int8 = 1  // 通过 - 申请已通过
    REFUSE  int8 = 2  // 拒绝 - 申请被拒绝
    BLACK   int8 = 3  // 拉黑 - 申请被拉黑
)