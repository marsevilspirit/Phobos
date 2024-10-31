package client

// FailMode 定义了失败处理模式
type FailMode int

const (
	// Failover 模式表示在失败时尝试下一个候选者
	Failover FailMode = iota
	// Failfast 模式表示在遇到失败时立刻返回错误
	Failfast
	// Failtry 模式表示在失败时对同一个候选者重新尝试
	Failtry
)

// SelectMode 定义了选择处理模式
type SelectMode int

const (
	// RandomSelect 模式表示随机选择一个候选者
	RandomSelect SelectMode = iota
	// RoundRobin 模式表示轮询选择候选者
	RoundRobin
	// WeightedRoundRobin 模式表示加权轮询选择候选者
	WeightedRoundRobin
	// WeightedICMP 模式表示基于 ICMP 加权的选择模式
	WeightedICMP
	// ConsistentHash 模式表示一致性哈希选择候选者
	ConsistentHash
	// Closest 模式表示选择最接近的候选者
	Closest
)

// `[...]string` 的声明方式表明数组的长度是根据初始化时的元素个数自动推断的
var selectModeStrs = [...]string{
	"RandomSelect",
	"RoundRobin",
	"WeightedRoundRobin",
	"WeightedICMP",
	"ConsistentHash",
	"Closest",
}

func (s SelectMode) String() string {
	return selectModeStrs[s]
}
